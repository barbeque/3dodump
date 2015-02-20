Content in this document was mostly lifted from the documentation packaged with the [operafs implementation for Linux](http://www.stack.nl/~svdb/operafs/). Some changes were made for clarity in order to aid future developers.

# OperaFS overview
OperaFS is the file system used for 3DO Multiplayer game CD-ROMs. One of the primary intents of the filesystem appears to have been that directories and files can have many copies ("Avatars") on disc, minimizing the sector distance between frequently accessed items, and therefore seek time.

A lesser benefit is that the multiple copies also increases the integrity of the disc, allowing the machine to recover more easily from read errors by hunting for another, intact copy of the data you were looking for. This implementation always takes the first copy when extracting.

# Legend
The following data formats are:

 * u8 - Unsigned 8 bit (aka `byte`)
 * u32 - Unsigned 32 bit (aka `uint32`)
 * s32 - Signed 32 bit (aka `int32`)

# File System Description

## Volume header
The volume header describes the disc.

| Format | Number of Bytes | What is it? |
| ------ | --------------- | ----------- |
| u8     | 1               | Record type (seemingly always 1) |
| u8     | 5               | Sync bytes (appear to always be 5A) |
| u8     | 1               | Record version (seemingly always 1) |
| u8     | 1               | Volume flags |
| u8     | 32              | Volume comment, null-terminated padded ASCII |
| u8     | 32              | Volume label, null-terminated padded ASCII |
| u32    | 4               | Volume identifier |
| u32    | 4               | Block size |
| u32    | 4               | Block count on disc |
| u32    | 4               | Root directory identifier |
| u32    | 4               | Root directory size in blocks |
| u32    | 4               | Block size in root directory |
| u32    | 4               | Number of copies of root directory (up to 7) |
| u32    | 4               | Root directory number one block location |
| u32    | 4 * 7           | Root directory copy block locations |

Once you have the volume header information, you can jump to a root directory definition by taking the first root directory block location specified in the header and multiplying it by the block size to get an offset in bytes.

Seek to that offset and you should find...

## Directory header
The directory header describes a directory on the disc.

| Format | Number of Bytes | What is it? |
| ------ | --------------- | ----------- |
| s32    | 4               | Next block in this directory (-1 if this is the last block) |
| s32    | 4               | Previous block in this directory (-1 if this is the first block) |
| u32    | 4               | Flags |
| u32    | 4               | Offset in bytes from the start of this header to the end of the directory's contents |
| u32    | 4               | Offset in bytes from the start of this header to the first directory entry in this block (apparently always 20 bytes) |

Jump off the end of this header and you should find the first item in the directory. Keep reading entries in until you've read the specified number of bytes, and this directory has been walked.

## Directory entry
The directory entry structure defines something that lives in a directory - for instance, a file or another directory.

| Format | Number of Bytes | What is it? |
| ------ | --------------- | ----------- |
| u32    | 4               | Flags (see below) |
| u32    | 4               | Identifier |
| u8     | 4               | Entry type, 4-byte ASCII, no null termination (see below) |
| u32    | 4               | Block size (seems same as other headers, possible sanity-check here) |
| u32    | 4               | Length of entry in bytes |
| u32    | 4               | Length of entry in blocks |
| u32    | 4               | Burst (purpose unknown) |
| u32    | 4               | Gap (purpose unknown) |
| u8     | 32              | File name, null-terminated ASCII |
| u32    | 4               | Number of copies of this entry's real data |
| u32    | 4*n             | Variable length structure - offset to original + copies from beginning of disc in blocks. For directories (type *dir) this points to the HEADER of the directory, not the start of the entries (which begin +20 bytes later) |

### Directory entry flags
The flag structure appears to contain information on both the least significant bit (LSB) and a mask to aid in directory iteration.

 * LSB
  * `0x02` - File
  * `0x06` - Special file
  * `0x07` - Directory
 * OR flags
  * `0x4000000` - Last entry in the block
  * `0x8000000` - Last entry in the directory

### Directory entry types
This signifies the type of various items you'll find.

| Entry type | What is it?                   |
| ---------- | ----------------------------- |
| `*dir`     | Directory |
| `*lbl`     | Label (points to volume header) |
| `*zap`     | Catapult |
| Other      | File type - seems to be related to original Mac type/creator code |
