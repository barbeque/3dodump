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
  //"bufio"
)

func check(e error) {
  if e != nil {
    panic(e)
  }
}

func read_all_entries_from_directory(file io.Reader) []DirectoryEntryTuple {
  // Assuming reader has started at the HEADER for the directory.
  // Scan ahead 20 bytes.
  _ = read_directory_header(file) // throw it away.
  // Now read directory entries until the cows come home
  entry, blobs := read_directory_entry(file)
  result := []DirectoryEntryTuple{ DirectoryEntryTuple{ Entry: entry, BlobPointers: blobs}}

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

  actual_root := read_directory_header(f)
  print_directory_header(actual_root, "Root directory")

  // The offset to first directory entry is almost always 0x14 - 20. The size of the header.
  // So just keep reading.
  first_entry, first_entry_blobs := read_directory_entry(f)
  print_directory_entry(first_entry, "First entry in root directory")
  print_blobs(first_entry_blobs)

  // Eat another file... is it really this easy?
  second_entry, second_entry_blobs := read_directory_entry(f)
  print_directory_entry(second_entry,  "Second entry in root directory")
  print_blobs(second_entry_blobs)

  third_entry, third_entry_blobs := read_directory_entry(f)
  print_directory_entry(third_entry, "Third entry in root directory")
  print_blobs(third_entry_blobs)

  fourth_entry, fourth_entry_blobs := read_directory_entry(f)
  print_directory_entry(fourth_entry, "Fourth entry in root directory")
  print_blobs(fourth_entry_blobs)

  fifth_entry, fifth_entry_blobs := read_directory_entry(f)
  print_directory_entry(fifth_entry, "Fifth entry in root directory")
  print_blobs(fifth_entry_blobs)

  sixth_entry, sixth_entry_blobs := read_directory_entry(f)
  print_directory_entry(sixth_entry, "Sixth entry in root directory")
  print_blobs(sixth_entry_blobs)

  seventh_entry, seventh_entry_blobs := read_directory_entry(f)
  print_directory_entry(seventh_entry, "Seventh entry in root directory")
  print_blobs(seventh_entry_blobs)

  eighth_entry, eighth_entry_blobs := read_directory_entry(f)
  print_directory_entry(eighth_entry, "8th entry in root directory")
  print_blobs(eighth_entry_blobs)

  for i := 0; i < 2; i++ {
    next_entry, next_blobs := read_directory_entry(f)
    print_directory_entry(next_entry, strconv.Itoa(i + 9) + "th entry in root directory")
    print_blobs(next_blobs)
  }

  // Only 10 entries in this directory.. so let's see how many bytes we've covered
  final_byte_location, err := f.Seek(0, os.SEEK_CUR)
  fmt.Println("We are now at byte", final_byte_location)
  fmt.Println("(aka", (final_byte_location - directory_start), "bytes since the start of the root dir.)")

  // CONFIRMED: First unused byte is how you figure out how long a directory is.

  // OK, now that we've stumbled onto something useful, let's take an entry we know
  // is a directory entry and try to find its header.
  // third_entry is "IronManData" on our test ISO.
  _, err = f.Seek(int64(third_entry_blobs[0] * vh.BlockSize), os.SEEK_SET)
  check(err)

  // Read the entire IronManData directory, header and all
  entries := read_all_entries_from_directory(f)
  for i, tuple := range entries {
    print_directory_entry(tuple.Entry, "Number " + strconv.Itoa(i) + " entry in the IronManData directory")
  }
}
