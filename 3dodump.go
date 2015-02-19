package main
import (
  "encoding/binary"
  //"encoding/hex"
  //"reflect"
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
  NextBlockInThisDirectory int32
  PreviousBlockInThisDirectory int32
  DirectoryFlags uint32
  OffsetToFirstUnusedByte uint32
  OffsetToFirstDirectoryEntry uint32
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

  for _, copy_offset := range rdh.OffsetsOfCopies {
    fmt.Println("Offset", copy_offset)
  }

  // Find the root offset. It'll be the root directory's first copy offset * the size of a block
  directory_start := int64(rdh.OffsetsOfCopies[0] * vh.BlockSize)
  fmt.Println("Root directory should start at offset", directory_start)

  // Read from that offset. Note: binary is a little weird here.
  off, err := f.Seek(directory_start, os.SEEK_SET)
  if err != nil {
    fmt.Println("Error seeking", err)
  }
  fmt.Println("offset now", off)

  var actual_root DirectoryHeader
  err = binary.Read(f, binary.BigEndian, actual_root)
  fmt.Println("Next root block", actual_root.NextBlockInThisDirectory)
  fmt.Println("Prev. root block", actual_root.PreviousBlockInThisDirectory)
  fmt.Println("Root dir flags", actual_root.DirectoryFlags)
  fmt.Println("Offset to first free byte in root dir", actual_root.OffsetToFirstUnusedByte)
  fmt.Println("Actual root directory offset to first entry", actual_root.OffsetToFirstDirectoryEntry)
}
