package main
import (
  "encoding/binary"
  //"encoding/hex"
  //"reflect"
  //"bytes"
  "os"
  "fmt"
  //"bufio"
)

func check(e error) {
  if e != nil {
    panic(e)
  }
}

type VolumeHeader struct {
  RecordType            byte
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

  var actual_root DirectoryHeader
  err = binary.Read(f, binary.BigEndian, &actual_root)
  check(err)
  fmt.Println("Directory header of root")
  fmt.Println("\tNext root block", actual_root.NextBlockInThisDirectory)
  fmt.Println("\tPrev. root block", actual_root.PreviousBlockInThisDirectory)
  fmt.Println("\tRoot dir flags", actual_root.DirectoryFlags)
  fmt.Println("\tOffset to first free byte in root dir", actual_root.OffsetToFirstUnusedByte)
  fmt.Println("\tOffset to first entry", actual_root.OffsetToFirstDirectoryEntry)

  // The offset to first directory entry is almost always 0x14 - 20. The size of the header.
  // So just keep reading.
  var first_entry DirectoryEntry
  err = binary.Read(f, binary.BigEndian, &first_entry)
  check(err)
  fmt.Println("Entry for first item in root directory")
  fmt.Println("\tFlags", first_entry.Flags)
  fmt.Println("\tIdentifier", first_entry.Identifier)
  fmt.Println("\tEntry Type", string(first_entry.EntryType[:]))
  fmt.Println("\tBlock size", first_entry.BlockSize) // Why is this duplicated everywhere??
  fmt.Println("\tFile name", string(first_entry.FileName[:]))
  fmt.Println("\tLength in bytes", first_entry.ByteLength)
  fmt.Println("\tLength in blocks", first_entry.BlockLength)
  fmt.Println("\tNumber of copies", first_entry.NumberOfCopies)
  // Read in the 'actual data' offset... see what it looks like
  var blob_address uint32
  binary.Read(f, binary.BigEndian, &blob_address)
  fmt.Println("\tData blob starts at block", blob_address)

  fmt.Println()

  // Eat another file... is it really this easy?
  var second_entry DirectoryEntry
  err = binary.Read(f, binary.BigEndian, &second_entry)
  check(err)
  fmt.Println("Entry for second item in root directory")
  fmt.Println("\tFlags", second_entry.Flags)
  fmt.Println("\tIdentifier", second_entry.Identifier)
  fmt.Println("\tEntry Type", string(second_entry.EntryType[:]))
  fmt.Println("\tBlock size", second_entry.BlockSize) // Why is this duplicated everywhere??
  fmt.Println("\tFile name", string(second_entry.FileName[:]))
  fmt.Println("\tLength in bytes", second_entry.ByteLength)
  fmt.Println("\tLength in blocks", second_entry.BlockLength)
  fmt.Println("\tNumber of copies", second_entry.NumberOfCopies)
}
