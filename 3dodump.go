package main
import (
  "encoding/binary"
  //"encoding/hex"
  //"reflect"
  //"bytes"
  "os"
  "fmt"
  "strconv"
  "io"
  "strings"
  //"bufio"
)

func check(e error) {
  if e != nil {
    panic(e)
  }
}

func read_all_entries_from_directory(file io.Reader) DirectoryIterationResult {
  // Assuming reader has started at the HEADER for the directory.
  // Scan ahead 20 bytes.
  _ = read_directory_header(file) // throw it away.
  // Now read directory entries until the cows come home
  entry, blobs := read_directory_entry(file)
  result := DirectoryIterationResult{ DirectoryEntryTuple{ Entry: entry, BlobPointers: blobs}}

  for !flag_is_last_entry_in_directory(entry.Flags) {
    entry, blobs = read_directory_entry(file)
    result = append(result, DirectoryEntryTuple{ Entry: entry, BlobPointers: blobs })
  }

  return result
}

func flag_is_last_entry_in_directory(flag uint32) bool {
  return flag & 0x80000000 != 0
}

func read_directory_header(file io.Reader) DirectoryHeader {
  var header DirectoryHeader
  err := binary.Read(file, binary.BigEndian, &header)
  check(err)
  return header
}

func read_directory_entry(file io.Reader) (DirectoryEntry, []uint32) {
  var entry DirectoryEntry
  err := binary.Read(file, binary.BigEndian, &entry)
  check(err)
  fmt.Println("Read directory entry for", string(entry.FileName[:]))

  // Blobs are separate from the entry since they are of
  // variable length and I'm not sure how to tell binary.Read about that
  // right now.
  total_blob_count := int(entry.NumberOfCopies) + 1

  if total_blob_count > 10 {
    fmt.Println("Error, provided blob pointer count (", total_blob_count, ") is ridiculous. You must not be reading a directory entry.")
  }

  blobs := make([]uint32, total_blob_count)

  // Read the blob pointers - copies AND the original.
  for i := 0; i < total_blob_count; i++ {
    binary.Read(file, binary.BigEndian, &blobs[i])
  }

  return entry, blobs
}

func print_directory_header(header DirectoryHeader, name string) {
  fmt.Print("Directory header for '", name , "'\n")
  fmt.Println("\tNext block", header.NextBlockInThisDirectory)
  fmt.Println("\tPrev. block", header.PreviousBlockInThisDirectory)
  fmt.Println("\tRoot dir flags", header.DirectoryFlags)
  fmt.Println("\tOffset to first free byte in dir", header.OffsetToFirstUnusedByte)
  fmt.Println("\tOffset to first entry", header.OffsetToFirstDirectoryEntry)
}

func print_directory_entry(entry DirectoryEntry, name string) {
  fmt.Print("Directory entry '", name, "'\n")
  fmt.Println("\tFlags", entry.Flags)
  fmt.Println("\tIdentifier", entry.Identifier)
  fmt.Println("\tEntry Type", string(entry.EntryType[:]))
  fmt.Println("\tBlock size", entry.BlockSize) // Why is this duplicated everywhere??
  fmt.Println("\tFile name", string(entry.FileName[:]))
  fmt.Println("\tLength in bytes", entry.ByteLength)
  fmt.Println("\tLength in blocks", entry.BlockLength)
  fmt.Println("\tNumber of copies", entry.NumberOfCopies)


  if entry.Flags & 0x40000000 != 0 {
    fmt.Println("\t\tLast entry in block")
  }
  if flag_is_last_entry_in_directory(entry.Flags) {
    fmt.Println("\t\tLast entry in directory")
  }

  fmt.Println()
}

func print_blobs(blob_block_pointers []uint32) {
  for i, b := range blob_block_pointers {
    fmt.Println("\tBlob", i, "at block", b)
  }
}

type VolumeHeader struct {
  RecordType            byte      // Must use PascalCase to get 'exported'.
  SynchronizationBytes  [5]byte
  RecordVersion byte
  VolumeFlags byte
  VolumeComment [32]byte
  VolumeLabel [32]byte
  VolumeIdentifier uint32
  BlockSize uint32
  BlockCount uint32
}

type RootDirectoryHeader struct {
  DirectoryIdentifier  uint32
  RootBlockCount       uint32
  RootBlockSize   uint32
  NumberOfCopies      uint32
  OffsetsOfCopies     [8]uint32
}

type DirectoryHeader struct {
  NextBlockInThisDirectory      int32
  PreviousBlockInThisDirectory  int32
  DirectoryFlags                uint32
  OffsetToFirstUnusedByte       uint32
  OffsetToFirstDirectoryEntry   uint32
}

type DirectoryEntry struct {
  Flags uint32
  Identifier uint32
  EntryType [4]byte
  BlockSize uint32
  ByteLength uint32
  BlockLength uint32
  Burst uint32
  Gap uint32
  FileName  [32]byte
  NumberOfCopies uint32
  // Offsets of copies is NumberOfCopies * uint32s
}

type DirectoryEntryTuple struct {
  Entry DirectoryEntry
  BlobPointers []uint32 // in blocks.
}

type FindError struct {
  s string
}
func (e *FindError) Error() string {
  return e.s
}

type DirectoryIterationResult []DirectoryEntryTuple
func (m DirectoryIterationResult) find_entry_by_name(name string) (DirectoryEntryTuple, error) {
  for _, e := range m {
    if clean_filename(e.Entry.FileName[:]) == name { // Could this be made shorter?
      return e, nil
    }
  }
  return DirectoryEntryTuple{}, &FindError{"Not found: '" + name + "'" }
}

func get_subdirectory(root DirectoryIterationResult, subdirectory_name string, blockSize uint32, f io.ReadSeeker) DirectoryIterationResult {
  entry_by_name, error := root.find_entry_by_name(subdirectory_name)
  check(error)
  _, error = f.Seek(int64(entry_by_name.BlobPointers[0] * blockSize), os.SEEK_SET)
  check(error)
  subdir_entries := read_all_entries_from_directory(f)
  return subdir_entries
}

func clean_filename(name_as_bytes []byte) string {
  return strings.TrimRight(string(name_as_bytes[:]), "\x00")
}

func extract_to_disk(entry DirectoryEntryTuple, blockSize uint32, f io.ReadSeeker) {
  block_location := entry.BlobPointers[0]
  byte_location := int64(block_location * blockSize)

  // Get current position in volume
  old_position, _ := f.Seek(0, os.SEEK_CUR)

  // Now move to where the file we want to extract is.
  f.Seek(byte_location, os.SEEK_SET)

  output_filename := "/tmp/" + clean_filename(entry.Entry.FileName[:])
  out, err := os.Create(output_filename)
  check(err)
  defer out.Close()

  for i := 0; i < int(entry.Entry.ByteLength); i++ {
    b := make([]byte, 1)
    _, err := f.Read(b)
    check(err)
    _, err = out.Write(b)
    check(err)
  }

  fmt.Println("Extracted to", output_filename)

  // Reset the position to where we were before in the volume
  f.Seek(old_position, os.SEEK_SET)
}

func main() {
  args := os.Args[1:]
  f, err := os.Open(args[0])
  check(err)
  defer f.Close()

  var vh VolumeHeader
  err = binary.Read(f, binary.BigEndian, &vh)
  fmt.Println("Record type", vh.RecordType)
  fmt.Println("Volume label", string(vh.VolumeLabel[:]))
  fmt.Println("Volume block size", vh.BlockSize)

  var rdh RootDirectoryHeader
  err = binary.Read(f, binary.BigEndian, &rdh)
  fmt.Println("Root id", rdh.DirectoryIdentifier)
  fmt.Println("# of copies", rdh.NumberOfCopies)
  fmt.Println("Root block size", rdh.RootBlockSize)

  fmt.Println("Root blocks can be found at:")
  for idx, copy_offset := range rdh.OffsetsOfCopies {
    fmt.Print("\tBlock ", copy_offset)
    if idx == 0 {
      fmt.Println(" (original)")
    } else {
      fmt.Println(" (copy)")
    }
  }

  // Find the root offset. It'll be the root directory's first copy offset * the size of a block
  directory_start := int64(rdh.OffsetsOfCopies[0] * vh.BlockSize)
  fmt.Println("Root directory should start at byte", directory_start)

  // Read from that offset. Note: binary is a little weird here.
  off, err := f.Seek(directory_start, os.SEEK_SET)
  if err != nil {
    fmt.Println("Error seeking", err)
  }
  fmt.Println("Seeked to", off)

  root_entries := read_all_entries_from_directory(f)
  for i, tuple := range root_entries {
    print_directory_entry(tuple.Entry, "Number " + strconv.Itoa(i) + " entry in the root directory")
  }

  // Read the IronManData directory
  iron_man_entries := get_subdirectory(root_entries, "IronManData", vh.BlockSize, f)
  for i, tuple := range iron_man_entries {
    print_directory_entry(tuple.Entry, "Number " + strconv.Itoa(i) + " entry in the IronManData directory")
  }

  // Then the QT directory under it
  qt_entries := get_subdirectory(iron_man_entries, "QT", vh.BlockSize, f)
  for i, tuple := range qt_entries {
    print_directory_entry(tuple.Entry, "Number " + strconv.Itoa(i) + " entry in the IronManData/QT directory")

    extract_to_disk(tuple, vh.BlockSize, f)
  }
}
