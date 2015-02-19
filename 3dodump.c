#include <stdio.h>
#include <stdlib.h>
#include <assert.h>

typedef unsigned char u8;
typedef unsigned int u32;
typedef signed int s32;

#define SWAP_UINT32(x) (((x) >> 24) | (((x) & 0x00FF0000) >> 8) | (((x) & 0x0000FF00) << 8) | ((x) << 24))

u8 next_u8(FILE* fp) {
	u8 result = 0;
	size_t read = fread(&result, 1, 1, fp);
	if(read != 1) {
		fprintf(stderr, "Could not read next byte\n");
		fprintf(stderr, "Failed at location %02lx\n", ftell(fp));
		exit(1);
	}
	return result;
}

u32 next_u32(FILE* fp) {
	u32 result = 0;
	size_t read = fread(&result, 4, 1, fp);
	if(read != 1) {
		fprintf(stderr, "Could not read next int (got %02lx)\n", read);
		fprintf(stderr, "Failed at location %02lx\n", ftell(fp));
		exit(1);
	}
	return SWAP_UINT32(result);
}

s32 next_s32(FILE* fp) {
	return (s32)next_u32(fp);
}

void print_as_binary(u8 byte) {
	for(unsigned int i = 0; i < 8; ++i) {
		printf(((byte & 1) << i != 0) ? "1" : "0");
	}
}

void skip(FILE* fp, int skip) {
	fseek(fp, skip, SEEK_CUR);
}

int main(int argc, char* argv[]) {
	FILE* fp = fopen(argv[1], "rb");
	if(!fp) {
		fprintf(stderr, "Could not open 3DO iso file at '%s'.\n", argv[1]);
		return -1;
	}

	u8 record_type = next_u8(fp);
	printf("Record type %02x\n", record_type);
	assert(record_type == 1);
	skip(fp, 5);
	u8 record_version = next_u8(fp);
	printf("Record version %02x\n", record_version);
	u8 volume_flags = next_u8(fp);
	printf("Volume flags ");
	print_as_binary(volume_flags);
	printf("\n");
	char* volume_comment = malloc(sizeof(char) * 32);
	fread(volume_comment, 1, 32, fp);
	printf("Volume comment '%s'\n", volume_comment);
	free(volume_comment); volume_comment = 0;
	char* volume_label = malloc(sizeof(char) * 32);
	fread(volume_label, 1, 32, fp);
	printf("Volume label '%s'\n", volume_label);
	free(volume_label); volume_label = 0;
	u32 volume_identifier = next_u32(fp);
	u32 block_size = next_u32(fp);
	u32 block_count = next_u32(fp);
	printf("Volume identifier %i, block size %i, block count %i\n", volume_identifier, block_size, block_count);
	u32 root_directory_id = next_u32(fp);
	u32 root_directory_block_count = next_u32(fp);
	u32 last_copy_of_root_directory = next_u32(fp);
	// 32 x u32 [locations of root directory copies]

	fclose(fp);
	return 0;
}
