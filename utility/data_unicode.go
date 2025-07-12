package utility

type DataCodingType int

const (
	DCT_Ascii_7_bit DataCodingType = iota
	DCT_Ascii_8_bit
	DCT_Binary
	DCT_UCS2
)

const (
	coding_group_bits_mask uint8 = 0xf0
	alphabet_mask          uint8 = 0x0c
	data_coding_mask       uint8 = 0x0f
	// message_class_mask uint8 = 0x10
	// message_class_number_mask uint8 = 0x03
	// compression_bit_mask uint8 = 0x20
	reserved_group_0                 uint8 = 0x80
	reserved_group_1                 uint8 = 0x90
	reserved_group_2                 uint8 = 0xa0
	reserved_group_3                 uint8 = 0xb0
	default_alphabet                 uint8 = 0x00
	data_8_bit                       uint8 = 0x04
	ucs2                             uint8 = 0x08
	reserved                         uint8 = 0x0c
	general_data_coding_indication_0 uint8 = 0x00
	general_data_coding_indication_1 uint8 = 0x10
	general_data_coding_indication_2 uint8 = 0x20
	general_data_coding_indication_3 uint8 = 0x30
	automatic_deletion_group_0       uint8 = 0x40
	automatic_deletion_group_1       uint8 = 0x50
	automatic_deletion_group_2       uint8 = 0x60
	automatic_deletion_group_3       uint8 = 0x70
	mwi_group_discard_message        uint8 = 0xc0
	mwi_group_store_message_1        uint8 = 0xd0
	mwi_group_store_message_2        uint8 = 0xe0
	data_coding_message_class        uint8 = 0xf0
)

func ExtractUnicode(data_coding int) DataCodingType {
	switch uint8(data_coding) & coding_group_bits_mask {
	case general_data_coding_indication_0:
		switch uint8(data_coding) & data_coding_mask {
		// ASCII
		case 0x00,
			0x01,
			0x03,
			0x05,
			0x06,
			0x07,
			0x0D,
			0x0E,
			0x0B,
			0x0C,
			0x0F:
			return DCT_Ascii_8_bit
		// Binary
		case 0x02, 0x04, 0x09, 0x0A:
			return DCT_Binary
		// usc2
		case 0x08:
			return DCT_UCS2
		default:
			return DCT_Ascii_7_bit
		}

	case general_data_coding_indication_1,
		general_data_coding_indication_2,
		general_data_coding_indication_3,
		automatic_deletion_group_0,
		automatic_deletion_group_1,
		automatic_deletion_group_2,
		automatic_deletion_group_3:
		switch uint8(data_coding) & alphabet_mask {
		case data_8_bit:
			return DCT_Binary
		case ucs2:
			return DCT_UCS2
		case default_alphabet,
			reserved:
			return DCT_Ascii_7_bit
		default:
			return DCT_Ascii_7_bit
		}

	case reserved_group_0,
		reserved_group_1,
		reserved_group_2,
		reserved_group_3,
		mwi_group_discard_message,
		mwi_group_store_message_1:
		return DCT_Ascii_7_bit
	case mwi_group_store_message_2:
		return DCT_UCS2
	case data_coding_message_class:
		// Bit2 Message coding 0->Default alphabet 1->8-bit data
		if (uint8(data_coding) & 0x04) != 0 {
			return DCT_Binary
		} else {
			return DCT_Ascii_7_bit
		}
	default:
		return DCT_Ascii_7_bit
	}
}
