
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package txscript

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"

	"golang.org/x/crypto/ripemd160"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

//操作码定义与txscript操作码相关的信息。如果是
//现在，是调用以对脚本执行操作码的函数。这个
//当前脚本作为切片传入，第一个成员是操作码
//本身。
type opcode struct {
	value  byte
	name   string
	length int
	opfunc func(*parsedOpcode, *Engine) error
}

//这些常量是BTC wiki上使用的官方操作码的值，
//比特币核心和大多数（如果不是的话）其他参考和软件相关
//处理BTC脚本。
const (
OP_0                   = 0x00 //零
OP_FALSE               = 0x00 //0 -阿卡OP0 0
OP_DATA_1              = 0x01 //一
OP_DATA_2              = 0x02 //二
OP_DATA_3              = 0x03 //三
OP_DATA_4              = 0x04 //四
OP_DATA_5              = 0x05 //五
OP_DATA_6              = 0x06 //六
OP_DATA_7              = 0x07 //七
OP_DATA_8              = 0x08 //八
OP_DATA_9              = 0x09 //九
OP_DATA_10             = 0x0a //十
OP_DATA_11             = 0x0b //十一
OP_DATA_12             = 0x0c //十二
OP_DATA_13             = 0x0d //十三
OP_DATA_14             = 0x0e //十四
OP_DATA_15             = 0x0f //十五
OP_DATA_16             = 0x10 //十六
OP_DATA_17             = 0x11 //十七
OP_DATA_18             = 0x12 //十八
OP_DATA_19             = 0x13 //十九
OP_DATA_20             = 0x14 //二十
OP_DATA_21             = 0x15 //二十一
OP_DATA_22             = 0x16 //二十二
OP_DATA_23             = 0x17 //二十三
OP_DATA_24             = 0x18 //二十四
OP_DATA_25             = 0x19 //二十五
OP_DATA_26             = 0x1a //二十六
OP_DATA_27             = 0x1b //二十七
OP_DATA_28             = 0x1c //二十八
OP_DATA_29             = 0x1d //二十九
OP_DATA_30             = 0x1e //三十
OP_DATA_31             = 0x1f //三十一
OP_DATA_32             = 0x20 //三十二
OP_DATA_33             = 0x21 //三十三
OP_DATA_34             = 0x22 //三十四
OP_DATA_35             = 0x23 //三十五
OP_DATA_36             = 0x24 //三十六
OP_DATA_37             = 0x25 //三十七
OP_DATA_38             = 0x26 //三十八
OP_DATA_39             = 0x27 //三十九
OP_DATA_40             = 0x28 //四十
OP_DATA_41             = 0x29 //四十一
OP_DATA_42             = 0x2a //四十二
OP_DATA_43             = 0x2b //四十三
OP_DATA_44             = 0x2c //四十四
OP_DATA_45             = 0x2d //四十五
OP_DATA_46             = 0x2e //四十六
OP_DATA_47             = 0x2f //四十七
OP_DATA_48             = 0x30 //四十八
OP_DATA_49             = 0x31 //四十九
OP_DATA_50             = 0x32 //五十
OP_DATA_51             = 0x33 //五十一
OP_DATA_52             = 0x34 //五十二
OP_DATA_53             = 0x35 //五十三
OP_DATA_54             = 0x36 //五十四
OP_DATA_55             = 0x37 //五十五
OP_DATA_56             = 0x38 //五十六
OP_DATA_57             = 0x39 //五十七
OP_DATA_58             = 0x3a //五十八
OP_DATA_59             = 0x3b //五十九
OP_DATA_60             = 0x3c //六十
OP_DATA_61             = 0x3d //六十一
OP_DATA_62             = 0x3e //六十二
OP_DATA_63             = 0x3f //六十三
OP_DATA_64             = 0x40 //六十四
OP_DATA_65             = 0x41 //六十五
OP_DATA_66             = 0x42 //六十六
OP_DATA_67             = 0x43 //六十七
OP_DATA_68             = 0x44 //六十八
OP_DATA_69             = 0x45 //六十九
OP_DATA_70             = 0x46 //七十
OP_DATA_71             = 0x47 //七十一
OP_DATA_72             = 0x48 //七十二
OP_DATA_73             = 0x49 //七十三
OP_DATA_74             = 0x4a //七十四
OP_DATA_75             = 0x4b //七十五
OP_PUSHDATA1           = 0x4c //七十六
OP_PUSHDATA2           = 0x4d //七十七
OP_PUSHDATA4           = 0x4e //七十八
OP_1NEGATE             = 0x4f //七十九
OP_RESERVED            = 0x50 //八十
OP_1                   = 0x51 //81-又名Op_真
OP_TRUE                = 0x51 //八十一
OP_2                   = 0x52 //八十二
OP_3                   = 0x53 //八十三
OP_4                   = 0x54 //八十四
OP_5                   = 0x55 //八十五
OP_6                   = 0x56 //八十六
OP_7                   = 0x57 //八十七
OP_8                   = 0x58 //八十八
OP_9                   = 0x59 //八十九
OP_10                  = 0x5a //九十
OP_11                  = 0x5b //九十一
OP_12                  = 0x5c //九十二
OP_13                  = 0x5d //九十三
OP_14                  = 0x5e //九十四
OP_15                  = 0x5f //九十五
OP_16                  = 0x60 //九十六
OP_NOP                 = 0x61 //九十七
OP_VER                 = 0x62 //九十八
OP_IF                  = 0x63 //九十九
OP_NOTIF               = 0x64 //一百
OP_VERIF               = 0x65 //一百零一
OP_VERNOTIF            = 0x66 //一百零二
OP_ELSE                = 0x67 //一百零三
OP_ENDIF               = 0x68 //一百零四
OP_VERIFY              = 0x69 //一百零五
OP_RETURN              = 0x6a //一百零六
OP_TOALTSTACK          = 0x6b //一百零七
OP_FROMALTSTACK        = 0x6c //一百零八
OP_2DROP               = 0x6d //一百零九
OP_2DUP                = 0x6e //一百一十
OP_3DUP                = 0x6f //一百一十一
OP_2OVER               = 0x70 //一百一十二
OP_2ROT                = 0x71 //一百一十三
OP_2SWAP               = 0x72 //一百一十四
OP_IFDUP               = 0x73 //一百一十五
OP_DEPTH               = 0x74 //一百一十六
OP_DROP                = 0x75 //一百一十七
OP_DUP                 = 0x76 //一百一十八
OP_NIP                 = 0x77 //一百一十九
OP_OVER                = 0x78 //一百二十
OP_PICK                = 0x79 //一百二十一
OP_ROLL                = 0x7a //一百二十二
OP_ROT                 = 0x7b //一百二十三
OP_SWAP                = 0x7c //一百二十四
OP_TUCK                = 0x7d //一百二十五
OP_CAT                 = 0x7e //一百二十六
OP_SUBSTR              = 0x7f //一百二十七
OP_LEFT                = 0x80 //一百二十八
OP_RIGHT               = 0x81 //一百二十九
OP_SIZE                = 0x82 //一百三十
OP_INVERT              = 0x83 //一百三十一
OP_AND                 = 0x84 //一百三十二
OP_OR                  = 0x85 //一百三十三
OP_XOR                 = 0x86 //一百三十四
OP_EQUAL               = 0x87 //一百三十五
OP_EQUALVERIFY         = 0x88 //一百三十六
OP_RESERVED1           = 0x89 //一百三十七
OP_RESERVED2           = 0x8a //一百三十八
OP_1ADD                = 0x8b //一百三十九
OP_1SUB                = 0x8c //一百四十
OP_2MUL                = 0x8d //一百四十一
OP_2DIV                = 0x8e //一百四十二
OP_NEGATE              = 0x8f //一百四十三
OP_ABS                 = 0x90 //一百四十四
OP_NOT                 = 0x91 //一百四十五
OP_0NOTEQUAL           = 0x92 //一百四十六
OP_ADD                 = 0x93 //一百四十七
OP_SUB                 = 0x94 //一百四十八
OP_MUL                 = 0x95 //一百四十九
OP_DIV                 = 0x96 //一百五十
OP_MOD                 = 0x97 //一百五十一
OP_LSHIFT              = 0x98 //一百五十二
OP_RSHIFT              = 0x99 //一百五十三
OP_BOOLAND             = 0x9a //一百五十四
OP_BOOLOR              = 0x9b //一百五十五
OP_NUMEQUAL            = 0x9c //一百五十六
OP_NUMEQUALVERIFY      = 0x9d //一百五十七
OP_NUMNOTEQUAL         = 0x9e //一百五十八
OP_LESSTHAN            = 0x9f //一百五十九
OP_GREATERTHAN         = 0xa0 //一百六十
OP_LESSTHANOREQUAL     = 0xa1 //一百六十一
OP_GREATERTHANOREQUAL  = 0xa2 //一百六十二
OP_MIN                 = 0xa3 //一百六十三
OP_MAX                 = 0xa4 //一百六十四
OP_WITHIN              = 0xa5 //一百六十五
OP_RIPEMD160           = 0xa6 //一百六十六
OP_SHA1                = 0xa7 //一百六十七
OP_SHA256              = 0xa8 //一百六十八
OP_HASH160             = 0xa9 //一百六十九
OP_HASH256             = 0xaa //一百七十
OP_CODESEPARATOR       = 0xab //一百七十一
OP_CHECKSIG            = 0xac //一百七十二
OP_CHECKSIGVERIFY      = 0xad //一百七十三
OP_CHECKMULTISIG       = 0xae //一百七十四
OP_CHECKMULTISIGVERIFY = 0xaf //一百七十五
OP_NOP1                = 0xb0 //一百七十六
OP_NOP2                = 0xb1 //一百七十七
OP_CHECKLOCKTIMEVERIFY = 0xb1 //177-又名Op_Nop2
OP_NOP3                = 0xb2 //一百七十八
OP_CHECKSEQUENCEVERIFY = 0xb2 //178-又名Op-Nop3
OP_NOP4                = 0xb3 //一百七十九
OP_NOP5                = 0xb4 //一百八十
OP_NOP6                = 0xb5 //一百八十一
OP_NOP7                = 0xb6 //一百八十二
OP_NOP8                = 0xb7 //一百八十三
OP_NOP9                = 0xb8 //一百八十四
OP_NOP10               = 0xb9 //一百八十五
OP_UNKNOWN186          = 0xba //一百八十六
OP_UNKNOWN187          = 0xbb //一百八十七
OP_UNKNOWN188          = 0xbc //一百八十八
OP_UNKNOWN189          = 0xbd //一百八十九
OP_UNKNOWN190          = 0xbe //一百九十
OP_UNKNOWN191          = 0xbf //一百九十一
OP_UNKNOWN192          = 0xc0 //一百九十二
OP_UNKNOWN193          = 0xc1 //一百九十三
OP_UNKNOWN194          = 0xc2 //一百九十四
OP_UNKNOWN195          = 0xc3 //一百九十五
OP_UNKNOWN196          = 0xc4 //一百九十六
OP_UNKNOWN197          = 0xc5 //一百九十七
OP_UNKNOWN198          = 0xc6 //一百九十八
OP_UNKNOWN199          = 0xc7 //一百九十九
OP_UNKNOWN200          = 0xc8 //二百
OP_UNKNOWN201          = 0xc9 //二百零一
OP_UNKNOWN202          = 0xca //二百零二
OP_UNKNOWN203          = 0xcb //二百零三
OP_UNKNOWN204          = 0xcc //二百零四
OP_UNKNOWN205          = 0xcd //二百零五
OP_UNKNOWN206          = 0xce //二百零六
OP_UNKNOWN207          = 0xcf //二百零七
OP_UNKNOWN208          = 0xd0 //二百零八
OP_UNKNOWN209          = 0xd1 //二百零九
OP_UNKNOWN210          = 0xd2 //二百一十
OP_UNKNOWN211          = 0xd3 //二百一十一
OP_UNKNOWN212          = 0xd4 //二百一十二
OP_UNKNOWN213          = 0xd5 //二百一十三
OP_UNKNOWN214          = 0xd6 //二百一十四
OP_UNKNOWN215          = 0xd7 //二百一十五
OP_UNKNOWN216          = 0xd8 //二百一十六
OP_UNKNOWN217          = 0xd9 //二百一十七
OP_UNKNOWN218          = 0xda //二百一十八
OP_UNKNOWN219          = 0xdb //二百一十九
OP_UNKNOWN220          = 0xdc //二百二十
OP_UNKNOWN221          = 0xdd //二百二十一
OP_UNKNOWN222          = 0xde //二百二十二
OP_UNKNOWN223          = 0xdf //二百二十三
OP_UNKNOWN224          = 0xe0 //二百二十四
OP_UNKNOWN225          = 0xe1 //二百二十五
OP_UNKNOWN226          = 0xe2 //二百二十六
OP_UNKNOWN227          = 0xe3 //二百二十七
OP_UNKNOWN228          = 0xe4 //二百二十八
OP_UNKNOWN229          = 0xe5 //二百二十九
OP_UNKNOWN230          = 0xe6 //二百三十
OP_UNKNOWN231          = 0xe7 //二百三十一
OP_UNKNOWN232          = 0xe8 //二百三十二
OP_UNKNOWN233          = 0xe9 //二百三十三
OP_UNKNOWN234          = 0xea //二百三十四
OP_UNKNOWN235          = 0xeb //二百三十五
OP_UNKNOWN236          = 0xec //二百三十六
OP_UNKNOWN237          = 0xed //二百三十七
OP_UNKNOWN238          = 0xee //二百三十八
OP_UNKNOWN239          = 0xef //二百三十九
OP_UNKNOWN240          = 0xf0 //二百四十
OP_UNKNOWN241          = 0xf1 //二百四十一
OP_UNKNOWN242          = 0xf2 //二百四十二
OP_UNKNOWN243          = 0xf3 //二百四十三
OP_UNKNOWN244          = 0xf4 //二百四十四
OP_UNKNOWN245          = 0xf5 //二百四十五
OP_UNKNOWN246          = 0xf6 //二百四十六
OP_UNKNOWN247          = 0xf7 //二百四十七
OP_UNKNOWN248          = 0xf8 //二百四十八
OP_UNKNOWN249          = 0xf9 //二百四十九
OP_SMALLINTEGER        = 0xfa //250-比特币核心内部
OP_PUBKEYS             = 0xfb //251-比特币核心内部
OP_UNKNOWN252          = 0xfc //二百五十二
OP_PUBKEYHASH          = 0xfd //253-比特币核心内部
OP_PUBKEY              = 0xfe //254-比特币核心内部
OP_INVALIDOPCODE       = 0xff //255-比特币核心内部
)

//条件执行常量。
const (
	OpCondFalse = 0
	OpCondTrue  = 1
	OpCondSkip  = 2
)

//opcodearray保存有关所有可能操作码的详细信息，例如字节数
//操作码和任何相关的数据应采用其人可读的名称，以及
//处理程序函数。
var opcodeArray = [256]opcode{
//数据推送操作码。
	OP_FALSE:     {OP_FALSE, "OP_0", 1, opcodeFalse},
	OP_DATA_1:    {OP_DATA_1, "OP_DATA_1", 2, opcodePushData},
	OP_DATA_2:    {OP_DATA_2, "OP_DATA_2", 3, opcodePushData},
	OP_DATA_3:    {OP_DATA_3, "OP_DATA_3", 4, opcodePushData},
	OP_DATA_4:    {OP_DATA_4, "OP_DATA_4", 5, opcodePushData},
	OP_DATA_5:    {OP_DATA_5, "OP_DATA_5", 6, opcodePushData},
	OP_DATA_6:    {OP_DATA_6, "OP_DATA_6", 7, opcodePushData},
	OP_DATA_7:    {OP_DATA_7, "OP_DATA_7", 8, opcodePushData},
	OP_DATA_8:    {OP_DATA_8, "OP_DATA_8", 9, opcodePushData},
	OP_DATA_9:    {OP_DATA_9, "OP_DATA_9", 10, opcodePushData},
	OP_DATA_10:   {OP_DATA_10, "OP_DATA_10", 11, opcodePushData},
	OP_DATA_11:   {OP_DATA_11, "OP_DATA_11", 12, opcodePushData},
	OP_DATA_12:   {OP_DATA_12, "OP_DATA_12", 13, opcodePushData},
	OP_DATA_13:   {OP_DATA_13, "OP_DATA_13", 14, opcodePushData},
	OP_DATA_14:   {OP_DATA_14, "OP_DATA_14", 15, opcodePushData},
	OP_DATA_15:   {OP_DATA_15, "OP_DATA_15", 16, opcodePushData},
	OP_DATA_16:   {OP_DATA_16, "OP_DATA_16", 17, opcodePushData},
	OP_DATA_17:   {OP_DATA_17, "OP_DATA_17", 18, opcodePushData},
	OP_DATA_18:   {OP_DATA_18, "OP_DATA_18", 19, opcodePushData},
	OP_DATA_19:   {OP_DATA_19, "OP_DATA_19", 20, opcodePushData},
	OP_DATA_20:   {OP_DATA_20, "OP_DATA_20", 21, opcodePushData},
	OP_DATA_21:   {OP_DATA_21, "OP_DATA_21", 22, opcodePushData},
	OP_DATA_22:   {OP_DATA_22, "OP_DATA_22", 23, opcodePushData},
	OP_DATA_23:   {OP_DATA_23, "OP_DATA_23", 24, opcodePushData},
	OP_DATA_24:   {OP_DATA_24, "OP_DATA_24", 25, opcodePushData},
	OP_DATA_25:   {OP_DATA_25, "OP_DATA_25", 26, opcodePushData},
	OP_DATA_26:   {OP_DATA_26, "OP_DATA_26", 27, opcodePushData},
	OP_DATA_27:   {OP_DATA_27, "OP_DATA_27", 28, opcodePushData},
	OP_DATA_28:   {OP_DATA_28, "OP_DATA_28", 29, opcodePushData},
	OP_DATA_29:   {OP_DATA_29, "OP_DATA_29", 30, opcodePushData},
	OP_DATA_30:   {OP_DATA_30, "OP_DATA_30", 31, opcodePushData},
	OP_DATA_31:   {OP_DATA_31, "OP_DATA_31", 32, opcodePushData},
	OP_DATA_32:   {OP_DATA_32, "OP_DATA_32", 33, opcodePushData},
	OP_DATA_33:   {OP_DATA_33, "OP_DATA_33", 34, opcodePushData},
	OP_DATA_34:   {OP_DATA_34, "OP_DATA_34", 35, opcodePushData},
	OP_DATA_35:   {OP_DATA_35, "OP_DATA_35", 36, opcodePushData},
	OP_DATA_36:   {OP_DATA_36, "OP_DATA_36", 37, opcodePushData},
	OP_DATA_37:   {OP_DATA_37, "OP_DATA_37", 38, opcodePushData},
	OP_DATA_38:   {OP_DATA_38, "OP_DATA_38", 39, opcodePushData},
	OP_DATA_39:   {OP_DATA_39, "OP_DATA_39", 40, opcodePushData},
	OP_DATA_40:   {OP_DATA_40, "OP_DATA_40", 41, opcodePushData},
	OP_DATA_41:   {OP_DATA_41, "OP_DATA_41", 42, opcodePushData},
	OP_DATA_42:   {OP_DATA_42, "OP_DATA_42", 43, opcodePushData},
	OP_DATA_43:   {OP_DATA_43, "OP_DATA_43", 44, opcodePushData},
	OP_DATA_44:   {OP_DATA_44, "OP_DATA_44", 45, opcodePushData},
	OP_DATA_45:   {OP_DATA_45, "OP_DATA_45", 46, opcodePushData},
	OP_DATA_46:   {OP_DATA_46, "OP_DATA_46", 47, opcodePushData},
	OP_DATA_47:   {OP_DATA_47, "OP_DATA_47", 48, opcodePushData},
	OP_DATA_48:   {OP_DATA_48, "OP_DATA_48", 49, opcodePushData},
	OP_DATA_49:   {OP_DATA_49, "OP_DATA_49", 50, opcodePushData},
	OP_DATA_50:   {OP_DATA_50, "OP_DATA_50", 51, opcodePushData},
	OP_DATA_51:   {OP_DATA_51, "OP_DATA_51", 52, opcodePushData},
	OP_DATA_52:   {OP_DATA_52, "OP_DATA_52", 53, opcodePushData},
	OP_DATA_53:   {OP_DATA_53, "OP_DATA_53", 54, opcodePushData},
	OP_DATA_54:   {OP_DATA_54, "OP_DATA_54", 55, opcodePushData},
	OP_DATA_55:   {OP_DATA_55, "OP_DATA_55", 56, opcodePushData},
	OP_DATA_56:   {OP_DATA_56, "OP_DATA_56", 57, opcodePushData},
	OP_DATA_57:   {OP_DATA_57, "OP_DATA_57", 58, opcodePushData},
	OP_DATA_58:   {OP_DATA_58, "OP_DATA_58", 59, opcodePushData},
	OP_DATA_59:   {OP_DATA_59, "OP_DATA_59", 60, opcodePushData},
	OP_DATA_60:   {OP_DATA_60, "OP_DATA_60", 61, opcodePushData},
	OP_DATA_61:   {OP_DATA_61, "OP_DATA_61", 62, opcodePushData},
	OP_DATA_62:   {OP_DATA_62, "OP_DATA_62", 63, opcodePushData},
	OP_DATA_63:   {OP_DATA_63, "OP_DATA_63", 64, opcodePushData},
	OP_DATA_64:   {OP_DATA_64, "OP_DATA_64", 65, opcodePushData},
	OP_DATA_65:   {OP_DATA_65, "OP_DATA_65", 66, opcodePushData},
	OP_DATA_66:   {OP_DATA_66, "OP_DATA_66", 67, opcodePushData},
	OP_DATA_67:   {OP_DATA_67, "OP_DATA_67", 68, opcodePushData},
	OP_DATA_68:   {OP_DATA_68, "OP_DATA_68", 69, opcodePushData},
	OP_DATA_69:   {OP_DATA_69, "OP_DATA_69", 70, opcodePushData},
	OP_DATA_70:   {OP_DATA_70, "OP_DATA_70", 71, opcodePushData},
	OP_DATA_71:   {OP_DATA_71, "OP_DATA_71", 72, opcodePushData},
	OP_DATA_72:   {OP_DATA_72, "OP_DATA_72", 73, opcodePushData},
	OP_DATA_73:   {OP_DATA_73, "OP_DATA_73", 74, opcodePushData},
	OP_DATA_74:   {OP_DATA_74, "OP_DATA_74", 75, opcodePushData},
	OP_DATA_75:   {OP_DATA_75, "OP_DATA_75", 76, opcodePushData},
	OP_PUSHDATA1: {OP_PUSHDATA1, "OP_PUSHDATA1", -1, opcodePushData},
	OP_PUSHDATA2: {OP_PUSHDATA2, "OP_PUSHDATA2", -2, opcodePushData},
	OP_PUSHDATA4: {OP_PUSHDATA4, "OP_PUSHDATA4", -4, opcodePushData},
	OP_1NEGATE:   {OP_1NEGATE, "OP_1NEGATE", 1, opcode1Negate},
	OP_RESERVED:  {OP_RESERVED, "OP_RESERVED", 1, opcodeReserved},
	OP_TRUE:      {OP_TRUE, "OP_1", 1, opcodeN},
	OP_2:         {OP_2, "OP_2", 1, opcodeN},
	OP_3:         {OP_3, "OP_3", 1, opcodeN},
	OP_4:         {OP_4, "OP_4", 1, opcodeN},
	OP_5:         {OP_5, "OP_5", 1, opcodeN},
	OP_6:         {OP_6, "OP_6", 1, opcodeN},
	OP_7:         {OP_7, "OP_7", 1, opcodeN},
	OP_8:         {OP_8, "OP_8", 1, opcodeN},
	OP_9:         {OP_9, "OP_9", 1, opcodeN},
	OP_10:        {OP_10, "OP_10", 1, opcodeN},
	OP_11:        {OP_11, "OP_11", 1, opcodeN},
	OP_12:        {OP_12, "OP_12", 1, opcodeN},
	OP_13:        {OP_13, "OP_13", 1, opcodeN},
	OP_14:        {OP_14, "OP_14", 1, opcodeN},
	OP_15:        {OP_15, "OP_15", 1, opcodeN},
	OP_16:        {OP_16, "OP_16", 1, opcodeN},

//控制操作码。
	OP_NOP:                 {OP_NOP, "OP_NOP", 1, opcodeNop},
	OP_VER:                 {OP_VER, "OP_VER", 1, opcodeReserved},
	OP_IF:                  {OP_IF, "OP_IF", 1, opcodeIf},
	OP_NOTIF:               {OP_NOTIF, "OP_NOTIF", 1, opcodeNotIf},
	OP_VERIF:               {OP_VERIF, "OP_VERIF", 1, opcodeReserved},
	OP_VERNOTIF:            {OP_VERNOTIF, "OP_VERNOTIF", 1, opcodeReserved},
	OP_ELSE:                {OP_ELSE, "OP_ELSE", 1, opcodeElse},
	OP_ENDIF:               {OP_ENDIF, "OP_ENDIF", 1, opcodeEndif},
	OP_VERIFY:              {OP_VERIFY, "OP_VERIFY", 1, opcodeVerify},
	OP_RETURN:              {OP_RETURN, "OP_RETURN", 1, opcodeReturn},
	OP_CHECKLOCKTIMEVERIFY: {OP_CHECKLOCKTIMEVERIFY, "OP_CHECKLOCKTIMEVERIFY", 1, opcodeCheckLockTimeVerify},
	OP_CHECKSEQUENCEVERIFY: {OP_CHECKSEQUENCEVERIFY, "OP_CHECKSEQUENCEVERIFY", 1, opcodeCheckSequenceVerify},

//堆栈操作码。
	OP_TOALTSTACK:   {OP_TOALTSTACK, "OP_TOALTSTACK", 1, opcodeToAltStack},
	OP_FROMALTSTACK: {OP_FROMALTSTACK, "OP_FROMALTSTACK", 1, opcodeFromAltStack},
	OP_2DROP:        {OP_2DROP, "OP_2DROP", 1, opcode2Drop},
	OP_2DUP:         {OP_2DUP, "OP_2DUP", 1, opcode2Dup},
	OP_3DUP:         {OP_3DUP, "OP_3DUP", 1, opcode3Dup},
	OP_2OVER:        {OP_2OVER, "OP_2OVER", 1, opcode2Over},
	OP_2ROT:         {OP_2ROT, "OP_2ROT", 1, opcode2Rot},
	OP_2SWAP:        {OP_2SWAP, "OP_2SWAP", 1, opcode2Swap},
	OP_IFDUP:        {OP_IFDUP, "OP_IFDUP", 1, opcodeIfDup},
	OP_DEPTH:        {OP_DEPTH, "OP_DEPTH", 1, opcodeDepth},
	OP_DROP:         {OP_DROP, "OP_DROP", 1, opcodeDrop},
	OP_DUP:          {OP_DUP, "OP_DUP", 1, opcodeDup},
	OP_NIP:          {OP_NIP, "OP_NIP", 1, opcodeNip},
	OP_OVER:         {OP_OVER, "OP_OVER", 1, opcodeOver},
	OP_PICK:         {OP_PICK, "OP_PICK", 1, opcodePick},
	OP_ROLL:         {OP_ROLL, "OP_ROLL", 1, opcodeRoll},
	OP_ROT:          {OP_ROT, "OP_ROT", 1, opcodeRot},
	OP_SWAP:         {OP_SWAP, "OP_SWAP", 1, opcodeSwap},
	OP_TUCK:         {OP_TUCK, "OP_TUCK", 1, opcodeTuck},

//拼接操作码。
	OP_CAT:    {OP_CAT, "OP_CAT", 1, opcodeDisabled},
	OP_SUBSTR: {OP_SUBSTR, "OP_SUBSTR", 1, opcodeDisabled},
	OP_LEFT:   {OP_LEFT, "OP_LEFT", 1, opcodeDisabled},
	OP_RIGHT:  {OP_RIGHT, "OP_RIGHT", 1, opcodeDisabled},
	OP_SIZE:   {OP_SIZE, "OP_SIZE", 1, opcodeSize},

//位逻辑操作码。
	OP_INVERT:      {OP_INVERT, "OP_INVERT", 1, opcodeDisabled},
	OP_AND:         {OP_AND, "OP_AND", 1, opcodeDisabled},
	OP_OR:          {OP_OR, "OP_OR", 1, opcodeDisabled},
	OP_XOR:         {OP_XOR, "OP_XOR", 1, opcodeDisabled},
	OP_EQUAL:       {OP_EQUAL, "OP_EQUAL", 1, opcodeEqual},
	OP_EQUALVERIFY: {OP_EQUALVERIFY, "OP_EQUALVERIFY", 1, opcodeEqualVerify},
	OP_RESERVED1:   {OP_RESERVED1, "OP_RESERVED1", 1, opcodeReserved},
	OP_RESERVED2:   {OP_RESERVED2, "OP_RESERVED2", 1, opcodeReserved},

//与数字相关的操作码。
	OP_1ADD:               {OP_1ADD, "OP_1ADD", 1, opcode1Add},
	OP_1SUB:               {OP_1SUB, "OP_1SUB", 1, opcode1Sub},
	OP_2MUL:               {OP_2MUL, "OP_2MUL", 1, opcodeDisabled},
	OP_2DIV:               {OP_2DIV, "OP_2DIV", 1, opcodeDisabled},
	OP_NEGATE:             {OP_NEGATE, "OP_NEGATE", 1, opcodeNegate},
	OP_ABS:                {OP_ABS, "OP_ABS", 1, opcodeAbs},
	OP_NOT:                {OP_NOT, "OP_NOT", 1, opcodeNot},
	OP_0NOTEQUAL:          {OP_0NOTEQUAL, "OP_0NOTEQUAL", 1, opcode0NotEqual},
	OP_ADD:                {OP_ADD, "OP_ADD", 1, opcodeAdd},
	OP_SUB:                {OP_SUB, "OP_SUB", 1, opcodeSub},
	OP_MUL:                {OP_MUL, "OP_MUL", 1, opcodeDisabled},
	OP_DIV:                {OP_DIV, "OP_DIV", 1, opcodeDisabled},
	OP_MOD:                {OP_MOD, "OP_MOD", 1, opcodeDisabled},
	OP_LSHIFT:             {OP_LSHIFT, "OP_LSHIFT", 1, opcodeDisabled},
	OP_RSHIFT:             {OP_RSHIFT, "OP_RSHIFT", 1, opcodeDisabled},
	OP_BOOLAND:            {OP_BOOLAND, "OP_BOOLAND", 1, opcodeBoolAnd},
	OP_BOOLOR:             {OP_BOOLOR, "OP_BOOLOR", 1, opcodeBoolOr},
	OP_NUMEQUAL:           {OP_NUMEQUAL, "OP_NUMEQUAL", 1, opcodeNumEqual},
	OP_NUMEQUALVERIFY:     {OP_NUMEQUALVERIFY, "OP_NUMEQUALVERIFY", 1, opcodeNumEqualVerify},
	OP_NUMNOTEQUAL:        {OP_NUMNOTEQUAL, "OP_NUMNOTEQUAL", 1, opcodeNumNotEqual},
	OP_LESSTHAN:           {OP_LESSTHAN, "OP_LESSTHAN", 1, opcodeLessThan},
	OP_GREATERTHAN:        {OP_GREATERTHAN, "OP_GREATERTHAN", 1, opcodeGreaterThan},
	OP_LESSTHANOREQUAL:    {OP_LESSTHANOREQUAL, "OP_LESSTHANOREQUAL", 1, opcodeLessThanOrEqual},
	OP_GREATERTHANOREQUAL: {OP_GREATERTHANOREQUAL, "OP_GREATERTHANOREQUAL", 1, opcodeGreaterThanOrEqual},
	OP_MIN:                {OP_MIN, "OP_MIN", 1, opcodeMin},
	OP_MAX:                {OP_MAX, "OP_MAX", 1, opcodeMax},
	OP_WITHIN:             {OP_WITHIN, "OP_WITHIN", 1, opcodeWithin},

//密码操作码。
	OP_RIPEMD160:           {OP_RIPEMD160, "OP_RIPEMD160", 1, opcodeRipemd160},
	OP_SHA1:                {OP_SHA1, "OP_SHA1", 1, opcodeSha1},
	OP_SHA256:              {OP_SHA256, "OP_SHA256", 1, opcodeSha256},
	OP_HASH160:             {OP_HASH160, "OP_HASH160", 1, opcodeHash160},
	OP_HASH256:             {OP_HASH256, "OP_HASH256", 1, opcodeHash256},
	OP_CODESEPARATOR:       {OP_CODESEPARATOR, "OP_CODESEPARATOR", 1, opcodeCodeSeparator},
	OP_CHECKSIG:            {OP_CHECKSIG, "OP_CHECKSIG", 1, opcodeCheckSig},
	OP_CHECKSIGVERIFY:      {OP_CHECKSIGVERIFY, "OP_CHECKSIGVERIFY", 1, opcodeCheckSigVerify},
	OP_CHECKMULTISIG:       {OP_CHECKMULTISIG, "OP_CHECKMULTISIG", 1, opcodeCheckMultiSig},
	OP_CHECKMULTISIGVERIFY: {OP_CHECKMULTISIGVERIFY, "OP_CHECKMULTISIGVERIFY", 1, opcodeCheckMultiSigVerify},

//保留的操作码。
	OP_NOP1:  {OP_NOP1, "OP_NOP1", 1, opcodeNop},
	OP_NOP4:  {OP_NOP4, "OP_NOP4", 1, opcodeNop},
	OP_NOP5:  {OP_NOP5, "OP_NOP5", 1, opcodeNop},
	OP_NOP6:  {OP_NOP6, "OP_NOP6", 1, opcodeNop},
	OP_NOP7:  {OP_NOP7, "OP_NOP7", 1, opcodeNop},
	OP_NOP8:  {OP_NOP8, "OP_NOP8", 1, opcodeNop},
	OP_NOP9:  {OP_NOP9, "OP_NOP9", 1, opcodeNop},
	OP_NOP10: {OP_NOP10, "OP_NOP10", 1, opcodeNop},

//未定义的操作码。
	OP_UNKNOWN186: {OP_UNKNOWN186, "OP_UNKNOWN186", 1, opcodeInvalid},
	OP_UNKNOWN187: {OP_UNKNOWN187, "OP_UNKNOWN187", 1, opcodeInvalid},
	OP_UNKNOWN188: {OP_UNKNOWN188, "OP_UNKNOWN188", 1, opcodeInvalid},
	OP_UNKNOWN189: {OP_UNKNOWN189, "OP_UNKNOWN189", 1, opcodeInvalid},
	OP_UNKNOWN190: {OP_UNKNOWN190, "OP_UNKNOWN190", 1, opcodeInvalid},
	OP_UNKNOWN191: {OP_UNKNOWN191, "OP_UNKNOWN191", 1, opcodeInvalid},
	OP_UNKNOWN192: {OP_UNKNOWN192, "OP_UNKNOWN192", 1, opcodeInvalid},
	OP_UNKNOWN193: {OP_UNKNOWN193, "OP_UNKNOWN193", 1, opcodeInvalid},
	OP_UNKNOWN194: {OP_UNKNOWN194, "OP_UNKNOWN194", 1, opcodeInvalid},
	OP_UNKNOWN195: {OP_UNKNOWN195, "OP_UNKNOWN195", 1, opcodeInvalid},
	OP_UNKNOWN196: {OP_UNKNOWN196, "OP_UNKNOWN196", 1, opcodeInvalid},
	OP_UNKNOWN197: {OP_UNKNOWN197, "OP_UNKNOWN197", 1, opcodeInvalid},
	OP_UNKNOWN198: {OP_UNKNOWN198, "OP_UNKNOWN198", 1, opcodeInvalid},
	OP_UNKNOWN199: {OP_UNKNOWN199, "OP_UNKNOWN199", 1, opcodeInvalid},
	OP_UNKNOWN200: {OP_UNKNOWN200, "OP_UNKNOWN200", 1, opcodeInvalid},
	OP_UNKNOWN201: {OP_UNKNOWN201, "OP_UNKNOWN201", 1, opcodeInvalid},
	OP_UNKNOWN202: {OP_UNKNOWN202, "OP_UNKNOWN202", 1, opcodeInvalid},
	OP_UNKNOWN203: {OP_UNKNOWN203, "OP_UNKNOWN203", 1, opcodeInvalid},
	OP_UNKNOWN204: {OP_UNKNOWN204, "OP_UNKNOWN204", 1, opcodeInvalid},
	OP_UNKNOWN205: {OP_UNKNOWN205, "OP_UNKNOWN205", 1, opcodeInvalid},
	OP_UNKNOWN206: {OP_UNKNOWN206, "OP_UNKNOWN206", 1, opcodeInvalid},
	OP_UNKNOWN207: {OP_UNKNOWN207, "OP_UNKNOWN207", 1, opcodeInvalid},
	OP_UNKNOWN208: {OP_UNKNOWN208, "OP_UNKNOWN208", 1, opcodeInvalid},
	OP_UNKNOWN209: {OP_UNKNOWN209, "OP_UNKNOWN209", 1, opcodeInvalid},
	OP_UNKNOWN210: {OP_UNKNOWN210, "OP_UNKNOWN210", 1, opcodeInvalid},
	OP_UNKNOWN211: {OP_UNKNOWN211, "OP_UNKNOWN211", 1, opcodeInvalid},
	OP_UNKNOWN212: {OP_UNKNOWN212, "OP_UNKNOWN212", 1, opcodeInvalid},
	OP_UNKNOWN213: {OP_UNKNOWN213, "OP_UNKNOWN213", 1, opcodeInvalid},
	OP_UNKNOWN214: {OP_UNKNOWN214, "OP_UNKNOWN214", 1, opcodeInvalid},
	OP_UNKNOWN215: {OP_UNKNOWN215, "OP_UNKNOWN215", 1, opcodeInvalid},
	OP_UNKNOWN216: {OP_UNKNOWN216, "OP_UNKNOWN216", 1, opcodeInvalid},
	OP_UNKNOWN217: {OP_UNKNOWN217, "OP_UNKNOWN217", 1, opcodeInvalid},
	OP_UNKNOWN218: {OP_UNKNOWN218, "OP_UNKNOWN218", 1, opcodeInvalid},
	OP_UNKNOWN219: {OP_UNKNOWN219, "OP_UNKNOWN219", 1, opcodeInvalid},
	OP_UNKNOWN220: {OP_UNKNOWN220, "OP_UNKNOWN220", 1, opcodeInvalid},
	OP_UNKNOWN221: {OP_UNKNOWN221, "OP_UNKNOWN221", 1, opcodeInvalid},
	OP_UNKNOWN222: {OP_UNKNOWN222, "OP_UNKNOWN222", 1, opcodeInvalid},
	OP_UNKNOWN223: {OP_UNKNOWN223, "OP_UNKNOWN223", 1, opcodeInvalid},
	OP_UNKNOWN224: {OP_UNKNOWN224, "OP_UNKNOWN224", 1, opcodeInvalid},
	OP_UNKNOWN225: {OP_UNKNOWN225, "OP_UNKNOWN225", 1, opcodeInvalid},
	OP_UNKNOWN226: {OP_UNKNOWN226, "OP_UNKNOWN226", 1, opcodeInvalid},
	OP_UNKNOWN227: {OP_UNKNOWN227, "OP_UNKNOWN227", 1, opcodeInvalid},
	OP_UNKNOWN228: {OP_UNKNOWN228, "OP_UNKNOWN228", 1, opcodeInvalid},
	OP_UNKNOWN229: {OP_UNKNOWN229, "OP_UNKNOWN229", 1, opcodeInvalid},
	OP_UNKNOWN230: {OP_UNKNOWN230, "OP_UNKNOWN230", 1, opcodeInvalid},
	OP_UNKNOWN231: {OP_UNKNOWN231, "OP_UNKNOWN231", 1, opcodeInvalid},
	OP_UNKNOWN232: {OP_UNKNOWN232, "OP_UNKNOWN232", 1, opcodeInvalid},
	OP_UNKNOWN233: {OP_UNKNOWN233, "OP_UNKNOWN233", 1, opcodeInvalid},
	OP_UNKNOWN234: {OP_UNKNOWN234, "OP_UNKNOWN234", 1, opcodeInvalid},
	OP_UNKNOWN235: {OP_UNKNOWN235, "OP_UNKNOWN235", 1, opcodeInvalid},
	OP_UNKNOWN236: {OP_UNKNOWN236, "OP_UNKNOWN236", 1, opcodeInvalid},
	OP_UNKNOWN237: {OP_UNKNOWN237, "OP_UNKNOWN237", 1, opcodeInvalid},
	OP_UNKNOWN238: {OP_UNKNOWN238, "OP_UNKNOWN238", 1, opcodeInvalid},
	OP_UNKNOWN239: {OP_UNKNOWN239, "OP_UNKNOWN239", 1, opcodeInvalid},
	OP_UNKNOWN240: {OP_UNKNOWN240, "OP_UNKNOWN240", 1, opcodeInvalid},
	OP_UNKNOWN241: {OP_UNKNOWN241, "OP_UNKNOWN241", 1, opcodeInvalid},
	OP_UNKNOWN242: {OP_UNKNOWN242, "OP_UNKNOWN242", 1, opcodeInvalid},
	OP_UNKNOWN243: {OP_UNKNOWN243, "OP_UNKNOWN243", 1, opcodeInvalid},
	OP_UNKNOWN244: {OP_UNKNOWN244, "OP_UNKNOWN244", 1, opcodeInvalid},
	OP_UNKNOWN245: {OP_UNKNOWN245, "OP_UNKNOWN245", 1, opcodeInvalid},
	OP_UNKNOWN246: {OP_UNKNOWN246, "OP_UNKNOWN246", 1, opcodeInvalid},
	OP_UNKNOWN247: {OP_UNKNOWN247, "OP_UNKNOWN247", 1, opcodeInvalid},
	OP_UNKNOWN248: {OP_UNKNOWN248, "OP_UNKNOWN248", 1, opcodeInvalid},
	OP_UNKNOWN249: {OP_UNKNOWN249, "OP_UNKNOWN249", 1, opcodeInvalid},

//比特币核心内部使用操作码。这里定义的是完整性。
	OP_SMALLINTEGER: {OP_SMALLINTEGER, "OP_SMALLINTEGER", 1, opcodeInvalid},
	OP_PUBKEYS:      {OP_PUBKEYS, "OP_PUBKEYS", 1, opcodeInvalid},
	OP_UNKNOWN252:   {OP_UNKNOWN252, "OP_UNKNOWN252", 1, opcodeInvalid},
	OP_PUBKEYHASH:   {OP_PUBKEYHASH, "OP_PUBKEYHASH", 1, opcodeInvalid},
	OP_PUBKEY:       {OP_PUBKEY, "OP_PUBKEY", 1, opcodeInvalid},

	OP_INVALIDOPCODE: {OP_INVALIDOPCODE, "OP_INVALIDOPCODE", 1, opcodeInvalid},
}

//opcodeonelinerepls定义在执行
//单线拆卸。这样做是为了匹配引用的输出
//在不更改nicer full中的操作码名称的情况下实现
//拆卸。
var opcodeOnelineRepls = map[string]string{
	"OP_1NEGATE": "-1",
	"OP_0":       "0",
	"OP_1":       "1",
	"OP_2":       "2",
	"OP_3":       "3",
	"OP_4":       "4",
	"OP_5":       "5",
	"OP_6":       "6",
	"OP_7":       "7",
	"OP_8":       "8",
	"OP_9":       "9",
	"OP_10":      "10",
	"OP_11":      "11",
	"OP_12":      "12",
	"OP_13":      "13",
	"OP_14":      "14",
	"OP_15":      "15",
	"OP_16":      "16",
}

//parseDopcode表示已分析的操作码，其中包括
//与之相关的潜在数据。
type parsedOpcode struct {
	opcode *opcode
	data   []byte
}

//is disabled返回操作码是否被禁用，因此始终
//在指令流中看不到（即使被条件关闭）。
func (pop *parsedOpcode) isDisabled() bool {
	switch pop.opcode.value {
	case OP_CAT:
		return true
	case OP_SUBSTR:
		return true
	case OP_LEFT:
		return true
	case OP_RIGHT:
		return true
	case OP_INVERT:
		return true
	case OP_AND:
		return true
	case OP_OR:
		return true
	case OP_XOR:
		return true
	case OP_2MUL:
		return true
	case OP_2DIV:
		return true
	case OP_MUL:
		return true
	case OP_DIV:
		return true
	case OP_MOD:
		return true
	case OP_LSHIFT:
		return true
	case OP_RSHIFT:
		return true
	default:
		return false
	}
}

//always illegal返回操作码在传递时是否始终是非法的
//即使在未执行的分支（它不是
//巧合的是它们是有条件的）。
func (pop *parsedOpcode) alwaysIllegal() bool {
	switch pop.opcode.value {
	case OP_VERIF:
		return true
	case OP_VERNOTIF:
		return true
	default:
		return false
	}
}

//is conditional返回操作码是否为条件操作码
//执行时更改条件执行堆栈。
func (pop *parsedOpcode) isConditional() bool {
	switch pop.opcode.value {
	case OP_IF:
		return true
	case OP_NOTIF:
		return true
	case OP_ELSE:
		return true
	case OP_ENDIF:
		return true
	default:
		return false
	}
}

//checkminimadatapush返回当前数据推送是否使用
//表示它的最小操作码。例如，值15可以
//使用op_data_1 15（其他变体）进行推送；但是，op_15是
//表示相同值且仅为单字节的单个操作码
//两个字节。
func (pop *parsedOpcode) checkMinimalDataPush() error {
	data := pop.data
	dataLen := len(data)
	opcode := pop.opcode.value

	if dataLen == 0 && opcode != OP_0 {
		str := fmt.Sprintf("zero length data push is encoded with "+
			"opcode %s instead of OP_0", pop.opcode.name)
		return scriptError(ErrMinimalData, str)
	} else if dataLen == 1 && data[0] >= 1 && data[0] <= 16 {
		if opcode != OP_1+data[0]-1 {
//应该使用操作1。op16
			str := fmt.Sprintf("data push of the value %d encoded "+
				"with opcode %s instead of OP_%d", data[0],
				pop.opcode.name, data[0])
			return scriptError(ErrMinimalData, str)
		}
	} else if dataLen == 1 && data[0] == 0x81 {
		if opcode != OP_1NEGATE {
			str := fmt.Sprintf("data push of the value -1 encoded "+
				"with opcode %s instead of OP_1NEGATE",
				pop.opcode.name)
			return scriptError(ErrMinimalData, str)
		}
	} else if dataLen <= 75 {
		if int(opcode) != dataLen {
//应该直接推一下
			str := fmt.Sprintf("data push of %d bytes encoded "+
				"with opcode %s instead of OP_DATA_%d", dataLen,
				pop.opcode.name, dataLen)
			return scriptError(ErrMinimalData, str)
		}
	} else if dataLen <= 255 {
		if opcode != OP_PUSHDATA1 {
			str := fmt.Sprintf("data push of %d bytes encoded "+
				"with opcode %s instead of OP_PUSHDATA1",
				dataLen, pop.opcode.name)
			return scriptError(ErrMinimalData, str)
		}
	} else if dataLen <= 65535 {
		if opcode != OP_PUSHDATA2 {
			str := fmt.Sprintf("data push of %d bytes encoded "+
				"with opcode %s instead of OP_PUSHDATA2",
				dataLen, pop.opcode.name)
			return scriptError(ErrMinimalData, str)
		}
	}
	return nil
}

//print返回操作码的可读字符串表示形式以供使用
//在脚本反汇编中。
func (pop *parsedOpcode) print(oneline bool) string {
//参考实现一行反汇编代替操作码
//表示值（例如，op_0到op_16和op_1negate）
//原始值。但是，当不进行单线分解时，
//我们更喜欢显示实际的操作码名称。因此，只需更换
//设置单线标志时出现问题的操作码。
	opcodeName := pop.opcode.name
	if oneline {
		if replName, ok := opcodeOnelineRepls[opcodeName]; ok {
			opcodeName = replName
		}

//无需对非数据推送操作码执行任何操作。
		if pop.opcode.length == 1 {
			return opcodeName
		}

		return fmt.Sprintf("%x", pop.data)
	}

//无需对非数据推送操作码执行任何操作。
	if pop.opcode.length == 1 {
		return opcodeName
	}

//为op pushdata操作码添加长度。
	retString := opcodeName
	switch pop.opcode.length {
	case -1:
		retString += fmt.Sprintf(" 0x%02x", len(pop.data))
	case -2:
		retString += fmt.Sprintf(" 0x%04x", len(pop.data))
	case -4:
		retString += fmt.Sprintf(" 0x%08x", len(pop.data))
	}

	return fmt.Sprintf("%s 0x%02x", retString, pop.data)
}

//字节返回与编码的操作码相关联的任何数据。
//脚本。这用于从解析的操作码中解离脚本。
func (pop *parsedOpcode) bytes() ([]byte, error) {
	var retbytes []byte
	if pop.opcode.length > 0 {
		retbytes = make([]byte, 1, pop.opcode.length)
	} else {
		retbytes = make([]byte, 1, 1+len(pop.data)-
			pop.opcode.length)
	}

	retbytes[0] = pop.opcode.value
	if pop.opcode.length == 1 {
		if len(pop.data) != 0 {
			str := fmt.Sprintf("internal consistency error - "+
				"parsed opcode %s has data length %d when %d "+
				"was expected", pop.opcode.name, len(pop.data),
				0)
			return nil, scriptError(ErrInternal, str)
		}
		return retbytes, nil
	}
	nbytes := pop.opcode.length
	if pop.opcode.length < 0 {
		l := len(pop.data)
//只需硬编码就可以避免这里的复杂性。
		switch pop.opcode.length {
		case -1:
			retbytes = append(retbytes, byte(l))
			nbytes = int(retbytes[1]) + len(retbytes)
		case -2:
			retbytes = append(retbytes, byte(l&0xff),
				byte(l>>8&0xff))
			nbytes = int(binary.LittleEndian.Uint16(retbytes[1:])) +
				len(retbytes)
		case -4:
			retbytes = append(retbytes, byte(l&0xff),
				byte((l>>8)&0xff), byte((l>>16)&0xff),
				byte((l>>24)&0xff))
			nbytes = int(binary.LittleEndian.Uint32(retbytes[1:])) +
				len(retbytes)
		}
	}

	retbytes = append(retbytes, pop.data...)

	if len(retbytes) != nbytes {
		str := fmt.Sprintf("internal consistency error - "+
			"parsed opcode %s has data length %d when %d was "+
			"expected", pop.opcode.name, len(retbytes), nbytes)
		return nil, scriptError(ErrInternal, str)
	}

	return retbytes, nil
}

//*************************************
//操作码实现函数从这里开始。
//*************************************

//opcodedisabled是一个常见的处理程序，用于处理已禁用的操作码。它返回一个
//指示操作码被禁用的适当错误。虽然它会
//通常更合理地检测脚本是否包含任何已禁用的
//在执行初始解析步骤之前的操作码，共识规则
//指示脚本在程序计数器通过
//禁用的操作码（即使它们出现在未执行的分支中）。
func opcodeDisabled(op *parsedOpcode, vm *Engine) error {
	str := fmt.Sprintf("attempt to execute disabled opcode %s",
		op.opcode.name)
	return scriptError(ErrDisabledOpcode, str)
}

//opcodereserved是所有保留操作码的通用处理程序。它返回一个
//指示操作码被保留的适当错误。
func opcodeReserved(op *parsedOpcode, vm *Engine) error {
	str := fmt.Sprintf("attempt to execute reserved opcode %s",
		op.opcode.name)
	return scriptError(ErrReservedOpcode, str)
}

//opcodeinvalid是所有无效操作码的通用处理程序。它返回一个
//指示操作码无效的适当错误。
func opcodeInvalid(op *parsedOpcode, vm *Engine) error {
	str := fmt.Sprintf("attempt to execute invalid opcode %s",
		op.opcode.name)
	return scriptError(ErrReservedOpcode, str)
}

//opcodefalse将空数组推送到数据堆栈以表示false。注释
//当根据数字编码共识将0编码为数字时，
//规则，是空数组。
func opcodeFalse(op *parsedOpcode, vm *Engine) error {
	vm.dstack.PushByteArray(nil)
	return nil
}

//opcodepushdata是大多数推送操作码的常见处理程序
//原始数据（字节）到数据堆栈。
func opcodePushData(op *parsedOpcode, vm *Engine) error {
	vm.dstack.PushByteArray(op.data)
	return nil
}

//opcode1negate将编码为数字的-1推送到数据栈。
func opcode1Negate(op *parsedOpcode, vm *Engine) error {
	vm.dstack.PushInt(scriptNum(-1))
	return nil
}

//操作码是小整数数据推送操作码的常见处理程序。它
//按操作码所代表的数值（从1到16）
//到数据栈上。
func opcodeN(op *parsedOpcode, vm *Engine) error {
//操作码都是连续定义的，因此数值是
//差异。
	vm.dstack.PushInt(scriptNum((op.opcode.value - (OP_1 - 1))))
	return nil
}

//opcodenop是nop系列操作码的常见处理程序。作为名字
//意味着它通常不做任何事情，但是，当
//为选择操作码设置了禁止使用nops的标志。
func opcodeNop(op *parsedOpcode, vm *Engine) error {
	switch op.opcode.value {
	case OP_NOP1, OP_NOP4, OP_NOP5,
		OP_NOP6, OP_NOP7, OP_NOP8, OP_NOP9, OP_NOP10:
		if vm.hasFlag(ScriptDiscourageUpgradableNops) {
			str := fmt.Sprintf("OP_NOP%d reserved for soft-fork "+
				"upgrades", op.opcode.value-(OP_NOP1-1))
			return scriptError(ErrDiscourageUpgradableNOPs, str)
		}
	}
	return nil
}

//在脚本执行期间，如果
//设置了特定标志。如果是这样，为了消除额外的来源
//关于第0版目击证人程序的讨厌的延展性，我们现在
//要求如下：对于op-if和op-not-if，顶层堆栈项必须
//要么是空字节片，要么是[0x01]。否则，位于
//将弹出堆栈并将其解释为布尔值。
func popIfBool(vm *Engine) (bool, error) {
//当不处于见证执行模式时，不执行v0见证
//程序，或者未设置最小if标志，将顶部堆栈项弹出为
//正常的布尔。
	if !vm.isWitnessVersionActive(0) || !vm.hasFlag(ScriptVerifyMinimalIf) {
		return vm.dstack.PopBool()
	}

//此时，正在执行v0见证程序，并且
//如果设置了标志，则在顶部堆栈上强制附加约束
//项目。
	so, err := vm.dstack.PopByteArray()
	if err != nil {
		return false, err
	}

//顶部元素的长度必须至少为一。
	if len(so) > 1 {
		str := fmt.Sprintf("minimal if is active, top element MUST "+
			"have a length of at least, instead length is %v",
			len(so))
		return false, scriptError(ErrMinimalIf, str)
	}

//此外，如果长度为1，则值必须为0x01。
	if len(so) == 1 && so[0] != 0x01 {
		str := fmt.Sprintf("minimal if is active, top stack item MUST "+
			"be an empty byte array or 0x01, is instead: %v",
			so[0])
		return false, scriptError(ErrMinimalIf, str)
	}

	return asBool(so), nil
}

//opcodeif将数据堆栈上的顶部项视为布尔值并将其移除。
//
//根据是否
//布尔值为true，并且此if是否按顺序位于正在执行的分支上
//允许根据条件正确执行其他操作码
//逻辑。当布尔值为真时，将执行第一个分支（除非
//此操作码嵌套在未执行的分支中）。
//
//<expression>if[语句][其他[语句]]endif
//
//注意，与所有非条件操作码不同，即使在
//它在一个非执行分支上，因此维护了适当的嵌套。
//
//数据堆栈转换：…布尔> >…
//条件堆栈转换：[…]->[…OpCondValue
func opcodeIf(op *parsedOpcode, vm *Engine) error {
	condVal := OpCondFalse
	if vm.isBranchExecuting() {
		ok, err := popIfBool(vm)
		if err != nil {
			return err
		}

		if ok {
			condVal = OpCondTrue
		}
	} else {
		condVal = OpCondSkip
	}
	vm.condStack = append(vm.condStack, condVal)
	return nil
}

//opcodenotif将数据堆栈上的顶部项视为布尔值并移除
//它。
//
//根据是否
//布尔值为true，并且此if是否按顺序位于正在执行的分支上
//允许根据条件正确执行其他操作码
//逻辑。当布尔值为假时，将执行第一个分支（除非
//此操作码嵌套在未执行的分支中）。
//
//<expression>notif[语句][其他[语句]]endif
//
//注意，与所有非条件操作码不同，即使在
//它在一个非执行分支上，因此维护了适当的嵌套。
//
//数据堆栈转换：…布尔> >…
//条件堆栈转换：[…]->[…OpCondValue
func opcodeNotIf(op *parsedOpcode, vm *Engine) error {
	condVal := OpCondFalse
	if vm.isBranchExecuting() {
		ok, err := popIfBool(vm)
		if err != nil {
			return err
		}

		if !ok {
			condVal = OpCondTrue
		}
	} else {
		condVal = OpCondSkip
	}
	vm.condStack = append(vm.condStack, condVal)
	return nil
}

//opcodeelse为if/else/endif的另一半反转条件执行。
//
//如果没有匹配的op_if，则返回错误。
//
//条件堆栈转换：…操作条件值]->[…OpCONDVALY！
func opcodeElse(op *parsedOpcode, vm *Engine) error {
	if len(vm.condStack) == 0 {
		str := fmt.Sprintf("encountered opcode %s with no matching "+
			"opcode to begin conditional execution", op.opcode.name)
		return scriptError(ErrUnbalancedConditional, str)
	}

	conditionalIdx := len(vm.condStack) - 1
	switch vm.condStack[conditionalIdx] {
	case OpCondTrue:
		vm.condStack[conditionalIdx] = OpCondFalse
	case OpCondFalse:
		vm.condStack[conditionalIdx] = OpCondTrue
	case OpCondSkip:
//值不会在skip中更改，因为它指示此操作码
//嵌套在未执行的分支中。
	}
	return nil
}

//opcodeendif终止条件块，从
//条件执行堆栈。
//
//如果没有匹配的op_if，则返回错误。
//
//条件堆栈转换：…操作条件值]->[…]
func opcodeEndif(op *parsedOpcode, vm *Engine) error {
	if len(vm.condStack) == 0 {
		str := fmt.Sprintf("encountered opcode %s with no matching "+
			"opcode to begin conditional execution", op.opcode.name)
		return scriptError(ErrUnbalancedConditional, str)
	}

	vm.condStack = vm.condStack[:len(vm.condStack)-1]
	return nil
}

//AbstractVerify将数据堆栈上的顶级项作为布尔值进行检查，并
//验证其计算结果是否为真。如果没有，则返回错误。
//堆栈上的项，或当该项的计算结果为false时。在后一种情况下
//由于最重要的项目评估，验证失败
//若为false，返回的错误将使用传递的错误代码。
func abstractVerify(op *parsedOpcode, vm *Engine, c ErrorCode) error {
	verified, err := vm.dstack.PopBool()
	if err != nil {
		return err
	}

	if !verified {
		str := fmt.Sprintf("%s failed", op.opcode.name)
		return scriptError(c, str)
	}
	return nil
}

//opcodeverify以布尔值的形式检查数据堆栈上的顶级项，并
//验证其计算结果是否为真。如果没有，则返回错误。
func opcodeVerify(op *parsedOpcode, vm *Engine) error {
	return abstractVerify(op, vm, ErrVerify)
}

//opcodereturn返回一个适当的错误，因为它始终是一个错误
//从脚本中提前返回。
func opcodeReturn(op *parsedOpcode, vm *Engine) error {
	return scriptError(ErrEarlyReturn, "script returned early")
}

//VerifyLockTime是一个用于验证锁定时间的助手函数。
func verifyLockTime(txLockTime, threshold, lockTime int64) error {
//脚本和事务中的锁定时间必须相同
//类型。
	if !((txLockTime < threshold && lockTime < threshold) ||
		(txLockTime >= threshold && lockTime >= threshold)) {
		str := fmt.Sprintf("mismatched locktime types -- tx locktime "+
			"%d, stack locktime %d", txLockTime, lockTime)
		return scriptError(ErrUnsatisfiedLockTime, str)
	}

	if lockTime > txLockTime {
		str := fmt.Sprintf("locktime requirement not satisfied -- "+
			"locktime is greater than the transaction locktime: "+
			"%d > %d", lockTime, txLockTime)
		return scriptError(ErrUnsatisfiedLockTime, str)
	}

	return nil
}

//opcodechecklocktimeverify将数据堆栈上的顶级项与
//包含脚本签名的事务的LockTime字段
//正在验证事务输出是否可使用。中频旗
//未设置scriptVerifyCheckLockTimeVerify，代码将继续，就像op nop2一样
//被处决了。
func opcodeCheckLockTimeVerify(op *parsedOpcode, vm *Engine) error {
//如果未设置scriptVerifyCheckLockTimeVerify脚本标志，请处理
//操作码改为op nop2。
	if !vm.hasFlag(ScriptVerifyCheckLockTimeVerify) {
		if vm.hasFlag(ScriptDiscourageUpgradableNops) {
			return scriptError(ErrDiscourageUpgradableNOPs,
				"OP_NOP2 reserved for soft-fork upgrades")
		}
		return nil
	}

//当前事务锁定时间是一个uint32，最大值为
//锁定时间为2^32-1（2106年）。但是，ScriptNum已签名
//因此，标准的4字节scriptnum最多只支持
//最大值为2^31-1（2038年）。因此，使用5字节的scriptnum
//因为它最多支持2^39-1，允许超过
//当前锁定时间限制。
//
//这里使用PeekBytearray而不是PeekInt，因为我们不想
//由于上述原因限制为4字节整数。
	so, err := vm.dstack.PeekByteArray(0)
	if err != nil {
		return err
	}
	lockTime, err := makeScriptNum(so, vm.dstack.verifyMinimalData, 5)
	if err != nil {
		return err
	}

//在极少数情况下，由于某些原因，参数需要小于0
//先做算术，你总是可以用
//0 op_max op_checklocktime验证。
	if lockTime < 0 {
		str := fmt.Sprintf("negative lock time: %d", lockTime)
		return scriptError(ErrNegativeLockTime, str)
	}

//事务的锁定时间字段要么是块高度，
//哪个事务已完成或时间戳取决于
//值在txscript.lockTimeThreshold之前。当它在
//门槛是一个街区的高度。
	err = verifyLockTime(int64(vm.tx.LockTime), LockTimeThreshold,
		int64(lockTime))
	if err != nil {
		return err
	}

//也可以禁用锁定时间功能，从而绕过
//如果每个事务输入都已由
//将其序列设置为最大值（Wire.MaxTxInSequenceNum）。这个
//条件将导致交易被允许进入区块链
//使操作码无效。
//
//通过强制输入被
//操作码解锁（其序列号小于最大值
//价值）。这足以证明正确性而无需
//检查每个输入。
//
//注意：这意味着即使交易由于
//另一个输入被解锁，当
//操作码使用的输入被锁定。
	if vm.tx.TxIn[vm.txIdx].Sequence == wire.MaxTxInSequenceNum {
		return scriptError(ErrUnsatisfiedLockTime,
			"transaction input is finalized")
	}

	return nil
}

//opcodechecksequenceverify将数据堆栈上的顶级项与
//包含脚本签名的事务的LockTime字段
//正在验证事务输出是否可使用。中频旗
//未设置scriptVerifyCheckSequenceVerify，代码将继续，就像op nop3一样
//被处决了。
func opcodeCheckSequenceVerify(op *parsedOpcode, vm *Engine) error {
//如果未设置scriptVerifyCheckSequenceVerify脚本标志，请处理
//操作码改为op nop3。
	if !vm.hasFlag(ScriptVerifyCheckSequenceVerify) {
		if vm.hasFlag(ScriptDiscourageUpgradableNops) {
			return scriptError(ErrDiscourageUpgradableNOPs,
				"OP_NOP3 reserved for soft-fork upgrades")
		}
		return nil
	}

//当前事务序列是一个uint32，导致
//序列2^32-1。但是，ScriptNum已签名，因此
//标准的4字节scriptnum最多只能支持
//2 ^ 31 -1。因此，这里使用5字节的scriptnum，因为它将支持
//最多2^39-1，允许超出当前序列的序列
//极限。
//
//这里使用PeekBytearray而不是PeekInt，因为我们不想
//由于上述原因限制为4字节整数。
	so, err := vm.dstack.PeekByteArray(0)
	if err != nil {
		return err
	}
	stackSequence, err := makeScriptNum(so, vm.dstack.verifyMinimalData, 5)
	if err != nil {
		return err
	}

//在极少数情况下，由于某些原因，参数需要小于0
//先做算术，你总是可以用
//0 op_max op_checksequenceverify。
	if stackSequence < 0 {
		str := fmt.Sprintf("negative sequence: %d", stackSequence)
		return scriptError(ErrNegativeLockTime, str)
	}

	sequence := int64(stackSequence)

//如果
//操作数设置了禁用的锁定时间标志，
//checkSequenceVerify的行为与nop相同。
	if sequence&int64(wire.SequenceLockTimeDisabled) != 0 {
		return nil
	}

//事务版本号不够高，无法触发csv规则
//失败了。
	if vm.tx.Version < 2 {
		str := fmt.Sprintf("invalid transaction version: %d",
			vm.tx.Version)
		return scriptError(ErrUnsatisfiedLockTime, str)
	}

//具有最高有效位集的序列号不是
//共识受限。测试交易的顺序
//数字没有此位集，禁止使用此属性
//要绕过checkSequenceVerify检查。
	txSequence := int64(vm.tx.TxIn[vm.txIdx].Sequence)
	if txSequence&int64(wire.SequenceLockTimeDisabled) != 0 {
		str := fmt.Sprintf("transaction sequence has sequence "+
			"locktime disabled bit set: 0x%x", txSequence)
		return scriptError(ErrUnsatisfiedLockTime, str)
	}

//在进行比较之前，屏蔽不一致的部分。
	lockTimeMask := int64(wire.SequenceLockTimeIsSeconds |
		wire.SequenceLockTimeMask)
	return verifyLockTime(txSequence&lockTimeMask,
		wire.SequenceLockTimeIsSeconds, sequence&lockTimeMask)
}

//opcodeToAltStack从主数据堆栈中移除顶部项并将其推送
//到备用数据堆栈。
//
//主数据栈转换：…x1 x2 x3]->[…X1 x2]
//alt数据堆栈转换：…y1 y2 y3]->[…Y1-Y2-Y3-x3]
func opcodeToAltStack(op *parsedOpcode, vm *Engine) error {
	so, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}
	vm.astack.PushByteArray(so)

	return nil
}

//opcodeFromAltStack从备用数据堆栈中移除顶项，并
//将其推送到主数据堆栈上。
//
//主数据栈转换：…x1 x2 x3]->[…X1 x2 x3 y3]
//alt数据堆栈转换：…y1 y2 y3]->[…Y1-Y2]
func opcodeFromAltStack(op *parsedOpcode, vm *Engine) error {
	so, err := vm.astack.PopByteArray()
	if err != nil {
		return err
	}
	vm.dstack.PushByteArray(so)

	return nil
}

//opcode2drop从数据堆栈中删除前2项。
//
//堆栈转换：…x1 x2 x3]->[…X1]
func opcode2Drop(op *parsedOpcode, vm *Engine) error {
	return vm.dstack.DropN(2)
}

//opcode2dup复制数据堆栈上的前2项。
//
//堆栈转换：…x1 x2 x3]->[…X1 x2 x3 x2 x3]
func opcode2Dup(op *parsedOpcode, vm *Engine) error {
	return vm.dstack.DupN(2)
}

//opcode3dup复制数据堆栈中前3项。
//
//堆栈转换：…x1 x2 x3]->[…x1 x2 x3 x1 x2 x3]
func opcode3Dup(op *parsedOpcode, vm *Engine) error {
	return vm.dstack.DupN(3)
}

//opcode2over复制了数据堆栈上前两项之前的两项。
//
//堆栈转换：…x1 x2 x3 x4]->[…x1 x2 x3 x4 x1 x2]
func opcode2Over(op *parsedOpcode, vm *Engine) error {
	return vm.dstack.OverN(2)
}

//opcode2rot将数据堆栈上的前6项向左旋转两次。
//
//堆栈转换：…x1 x2 x3 x4 x5 x6]->[…x3 x4 x5 x6 x1 x2]
func opcode2Rot(op *parsedOpcode, vm *Engine) error {
	return vm.dstack.RotN(2)
}

//opcode2swap将数据堆栈上的前2项与后面的2项交换
//在他们面前。
//
//堆栈转换：…x1 x2 x3 x4]->[…x3x4x1-x2]
func opcode2Swap(op *parsedOpcode, vm *Engine) error {
	return vm.dstack.SwapN(2)
}

//opcodeifdup如果不是零，则复制堆栈的顶部项。
//
//堆栈转换（x1==0）：[…X1] ->…X1]
//堆栈转换（x1！= 0）X1] ->…X1-X1]
func opcodeIfDup(op *parsedOpcode, vm *Engine) error {
	so, err := vm.dstack.PeekByteArray(0)
	if err != nil {
		return err
	}

//如果不是零，则推送数据副本
	if asBool(so) {
		vm.dstack.PushByteArray(so)
	}

	return nil
}

//在执行此操作之前，opcodeDepth会推送数据堆栈的深度。
//操作码，编码为数字，进入数据栈。
//
//堆栈转换：[…]->[…<num of items on the stack>]
//带有2个项目的示例：【x1 x2】->【x1 x2 2】
//带有3个项目的示例：【x1 x2 x3】->【x1 x2 x3 3】
func opcodeDepth(op *parsedOpcode, vm *Engine) error {
	vm.dstack.PushInt(scriptNum(vm.dstack.Depth()))
	return nil
}

//opcodeDrop从数据堆栈中删除顶部项。
//
//堆栈转换：…x1 x2 x3]->[…X1 x2]
func opcodeDrop(op *parsedOpcode, vm *Engine) error {
	return vm.dstack.DropN(1)
}

//opcodedup复制数据堆栈上的顶部项。
//
//堆栈转换：…x1 x2 x3]->[…X1 x2 x3 x3]
func opcodeDup(op *parsedOpcode, vm *Engine) error {
	return vm.dstack.DupN(1)
}

//opcodenip删除数据堆栈上顶部项之前的项。
//
//堆栈转换：…x1 x2 x3]->[…X1 x3]
func opcodeNip(op *parsedOpcode, vm *Engine) error {
	return vm.dstack.NipN(1)
}

//opcodeover复制数据堆栈上顶部项之前的项。
//
//堆栈转换：…x1 x2 x3]->[…X1 x2 x3 x2]
func opcodeOver(op *parsedOpcode, vm *Engine) error {
	return vm.dstack.OverN(1)
}

//opcodepick将数据堆栈上的顶部项视为整数并重复
//堆栈上的项，该项的数目返回顶部。
//
//堆栈转换：[xn…x2 x1 x0 n]->[xn…X2x1-x0xn]
//n=1的示例：【x2 x1 x0 1】->【x2 x1 x0 x1】
//n=2的示例：【x2 x1 x0 2】->【x2 x1 x0 x2】
func opcodePick(op *parsedOpcode, vm *Engine) error {
	val, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	return vm.dstack.PickN(val.Int32())
}

//opcoderoll将数据堆栈上的顶级项视为整数并移动
//堆栈上的项，该项的数目返回顶部。
//
//堆栈转换：[xn…x2 x1 x0 n]->[…X2x1-x0xn]
//n=1的示例：【x2 x1 x0 1】->【x2 x0 x1】
//n=2的示例：【x2 x1 x0 2】->【x1 x0 x2】
func opcodeRoll(op *parsedOpcode, vm *Engine) error {
	val, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	return vm.dstack.RollN(val.Int32())
}

//opcoderot将数据堆栈上的前3项向左旋转。
//
//堆栈转换：…x1 x2 x3]->[…X2 x3 x1]
func opcodeRot(op *parsedOpcode, vm *Engine) error {
	return vm.dstack.RotN(1)
}

//opcodeswap交换堆栈上的前两项。
//
//堆栈转换：…X1 x2] ->…X2X1]
func opcodeSwap(op *parsedOpcode, vm *Engine) error {
	return vm.dstack.SwapN(1)
}

//opcodetuck在
//第二项到第一项。
//
//堆栈转换：…X1 x2] ->…X2 x1 x2]
func opcodeTuck(op *parsedOpcode, vm *Engine) error {
	return vm.dstack.Tuck()
}

//opcodeSize将数据堆栈顶部项的大小推送到数据上
//栈。
//
//堆栈转换：…X1] ->…X1 LeN（X1）
func opcodeSize(op *parsedOpcode, vm *Engine) error {
	so, err := vm.dstack.PeekByteArray(0)
	if err != nil {
		return err
	}

	vm.dstack.PushInt(scriptNum(len(so)))
	return nil
}

//opcodeEqual删除数据堆栈的前2项，并将其作为原始项进行比较
//字节，并将编码为布尔值的结果推回到堆栈。
//
//堆栈转换：…X1 x2] ->…布尔
func opcodeEqual(op *parsedOpcode, vm *Engine) error {
	a, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}
	b, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}

	vm.dstack.PushBool(bytes.Equal(a, b))
	return nil
}

//opcodeEqualVerify是opcodeEqual和opcodeVerify的组合。
//具体来说，它会删除数据堆栈的前2个项，并对它们进行比较，
//并将编码为布尔值的结果推回到堆栈中。然后，它
//将数据堆栈上的顶级项作为布尔值检查并验证它
//计算结果为true。如果没有，则返回错误。
//
//堆栈转换：…X1 x2] ->…布尔> >…
func opcodeEqualVerify(op *parsedOpcode, vm *Engine) error {
	err := opcodeEqual(op, vm)
	if err == nil {
		err = abstractVerify(op, vm, ErrEqualVerify)
	}
	return err
}

//opcode1add将数据堆栈上的顶项视为整数并替换
//它的递增值（加1）。
//
//堆栈转换：…X1 x2] ->…X1 x2+ 1
func opcode1Add(op *parsedOpcode, vm *Engine) error {
	m, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	vm.dstack.PushInt(m + 1)
	return nil
}

//opcode1sub将数据堆栈上的顶部项作为整数处理并替换
//它的递减值（减1）。
//
//堆栈转换：…X1 x2] ->…X1-X2-1
func opcode1Sub(op *parsedOpcode, vm *Engine) error {
	m, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}
	vm.dstack.PushInt(m - 1)

	return nil
}

//opcodenegate将数据堆栈上的顶部项作为整数处理并替换
//它的否定。
//
//堆栈转换：…X1 x2] ->…X1-X2]
func opcodeNegate(op *parsedOpcode, vm *Engine) error {
	m, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	vm.dstack.PushInt(-m)
	return nil
}

//opcodeabs将数据堆栈上的顶项视为整数并替换它
//它的绝对值。
//
//堆栈转换：…X1 x2] ->…X1 ABS（X2）
func opcodeAbs(op *parsedOpcode, vm *Engine) error {
	m, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if m < 0 {
		m = -m
	}
	vm.dstack.PushInt(m)
	return nil
}

//opcodenot将数据堆栈上的第一项视为整数并替换
//它的“反转”值（0变为1，非零变为0）。
//
//注意：虽然将顶部项目视为
//布尔值，然后推相反的值，这就是这个的目的
//操作码是非常重要的，因为整数是
//与布尔值和此操作码的一致性规则解释不同
//指示项被解释为整数。
//
//堆栈转换（x2==0）：[…X1 0 ] ->…X1 1
//堆栈转换（x2！= 0）X1 1 ] ->…X1 0
//堆栈转换（x2！= 0）X1 17 ] ->…X1 0
func opcodeNot(op *parsedOpcode, vm *Engine) error {
	m, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if m == 0 {
		vm.dstack.PushInt(scriptNum(1))
	} else {
		vm.dstack.PushInt(scriptNum(0))
	}
	return nil
}

//opcode0notequal将数据堆栈上的顶级项视为整数，并
//如果为零，则替换为0；如果不为零，则替换为1。
//
//堆栈转换（x2==0）：[…X1 0 ] ->…X1 0
//堆栈转换（x2！= 0）X1 1 ] ->…X1 1
//堆栈转换（x2！= 0）X1 17 ] ->…X1 1
func opcode0NotEqual(op *parsedOpcode, vm *Engine) error {
	m, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if m != 0 {
		m = 1
	}
	vm.dstack.PushInt(m)
	return nil
}

//opcodeAdd将数据堆栈中前两项视为整数并替换
//和他们的总数。
//
//堆栈转换：…X1 x2] ->…X1+X2]
func opcodeAdd(op *parsedOpcode, vm *Engine) error {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	vm.dstack.PushInt(v0 + v1)
	return nil
}

//opcodeSub将数据堆栈上的前两项视为整数并替换
//从第二个项到顶部项减去顶部项的结果。
//条目。
//
//堆栈转换：…X1 x2] ->…X1-X2]
func opcodeSub(op *parsedOpcode, vm *Engine) error {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	vm.dstack.PushInt(v1 - v0)
	return nil
}

//opcodebooland将数据堆栈上的前两项视为整数。什么时候？
//它们都不是0，而是用1替换，否则是0。
//
//堆栈转换（x1==0，x2==0）：[…0 0“->…0
//堆栈转换（x1！=0，x2==0）：[…5 0“->…0
//堆栈转换（x1==0，x2！= 0）0 7“->…0
//堆栈转换（x1！= 0，X2！= 0）4 8“->…1
func opcodeBoolAnd(op *parsedOpcode, vm *Engine) error {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if v0 != 0 && v1 != 0 {
		vm.dstack.PushInt(scriptNum(1))
	} else {
		vm.dstack.PushInt(scriptNum(0))
	}

	return nil
}

//opcodeboolor将数据堆栈上的前两项视为整数。什么时候？
//它们中的任何一个都不是零，它们被替换为1，否则是0。
//
//堆栈转换（x1==0，x2==0）：[…0 0“->…0
//堆栈转换（x1！=0，x2==0）：[…5 0“->…1
//堆栈转换（x1==0，x2！= 0）0 7“->…1
//堆栈转换（x1！= 0，X2！= 0）4 8“->…1
func opcodeBoolOr(op *parsedOpcode, vm *Engine) error {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if v0 != 0 || v1 != 0 {
		vm.dstack.PushInt(scriptNum(1))
	} else {
		vm.dstack.PushInt(scriptNum(0))
	}

	return nil
}

//opcodeNumEqual将数据堆栈上的前两项视为整数。什么时候？
//它们相等，替换为1，否则替换为0。
//
//堆栈转换（x1==x2）：[…5 5“->…1
//堆栈转换（x1！= x2）：…5 7“->…0
func opcodeNumEqual(op *parsedOpcode, vm *Engine) error {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if v0 == v1 {
		vm.dstack.PushInt(scriptNum(1))
	} else {
		vm.dstack.PushInt(scriptNum(0))
	}

	return nil
}

//opcodeNumEqualVerify是opcodeNumEqual和opcodeVerify的组合。
//
//具体来说，将数据堆栈上的前两个项视为整数。什么时候？
//它们相等，替换为1，否则替换为0。然后，它检查
//数据堆栈上的顶级项作为布尔值并验证其计算结果
//成真。如果没有，则返回错误。
//
//堆栈转换：…X1 x2] ->…布尔> >…
func opcodeNumEqualVerify(op *parsedOpcode, vm *Engine) error {
	err := opcodeNumEqual(op, vm)
	if err == nil {
		err = abstractVerify(op, vm, ErrNumEqualVerify)
	}
	return err
}

//opcodeNumNotEqual将数据堆栈中前两个项视为整数。
//当它们不相等时，它们将替换为1，否则将替换为0。
//
//堆栈转换（x1==x2）：[…5 5“->…0
//堆栈转换（x1！= x2）：…5 7“->…1
func opcodeNumNotEqual(op *parsedOpcode, vm *Engine) error {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if v0 != v1 {
		vm.dstack.PushInt(scriptNum(1))
	} else {
		vm.dstack.PushInt(scriptNum(0))
	}

	return nil
}

//opcodelessthan将数据堆栈中前两项视为整数。什么时候？
//第二至第一项小于第一项，用1替换，
//否则为0。
//
//堆栈转换：…X1 x2] ->…布尔
func opcodeLessThan(op *parsedOpcode, vm *Engine) error {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if v1 < v0 {
		vm.dstack.PushInt(scriptNum(1))
	} else {
		vm.dstack.PushInt(scriptNum(0))
	}

	return nil
}

//opcodegreaterThan将数据堆栈中前两个项视为整数。
//当第二个到第一个项大于第一个项时，它们将被替换。
//使用1，否则为0。
//
//堆栈转换：…X1 x2] ->…布尔
func opcodeGreaterThan(op *parsedOpcode, vm *Engine) error {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if v1 > v0 {
		vm.dstack.PushInt(scriptNum(1))
	} else {
		vm.dstack.PushInt(scriptNum(0))
	}
	return nil
}

//opcodelsthanorequal将数据堆栈中前两项视为整数。
//当第二个到第一个项小于或等于第一个项时，它们是
//替换为1，否则为0。
//
//堆栈转换：…X1 x2] ->…布尔
func opcodeLessThanOrEqual(op *parsedOpcode, vm *Engine) error {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if v1 <= v0 {
		vm.dstack.PushInt(scriptNum(1))
	} else {
		vm.dstack.PushInt(scriptNum(0))
	}
	return nil
}

//opcodegreaterThanOrEqual将数据堆栈中前两项视为
//整数。当第二个到第一个项大于或等于顶部时
//项，它们将替换为1，否则将替换为0。
//
//堆栈转换：…X1 x2] ->…布尔
func opcodeGreaterThanOrEqual(op *parsedOpcode, vm *Engine) error {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if v1 >= v0 {
		vm.dstack.PushInt(scriptNum(1))
	} else {
		vm.dstack.PushInt(scriptNum(0))
	}

	return nil
}

//opcodemin将数据堆栈上的前两项视为整数并替换
//他们至少有两个。
//
//堆栈转换：…X1 x2] ->…min（x1，x2）
func opcodeMin(op *parsedOpcode, vm *Engine) error {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if v1 < v0 {
		vm.dstack.PushInt(v1)
	} else {
		vm.dstack.PushInt(v0)
	}

	return nil
}

//opcodemax将数据堆栈上的前两项视为整数并替换
//最多两个。
//
//堆栈转换：…X1 x2] ->…max（x1，x2）
func opcodeMax(op *parsedOpcode, vm *Engine) error {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if v1 > v0 {
		vm.dstack.PushInt(v1)
	} else {
		vm.dstack.PushInt(v0)
	}

	return nil
}

//opcodewithin将数据堆栈中前3项视为整数。当
//要测试的值在指定的范围内（包括左），它们是
//替换为1，否则为0。
//
//顶部项目是最大值，第二个顶部项目是最小值，以及
//第三项到第一项是要测试的值。
//
//堆栈转换：…x1最小最大值]->[…布尔
func opcodeWithin(op *parsedOpcode, vm *Engine) error {
	maxVal, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	minVal, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	x, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if x >= minVal && x < maxVal {
		vm.dstack.PushInt(scriptNum(1))
	} else {
		vm.dstack.PushInt(scriptNum(0))
	}
	return nil
}

//calchash计算buf上散列器的散列值。
func calcHash(buf []byte, hasher hash.Hash) []byte {
	hasher.Write(buf)
	return hasher.Sum(nil)
}

//opcoderipemd160将数据堆栈的顶部项视为原始字节，并
//将其替换为ripemd160（数据）。
//
//堆栈转换：…X1] ->…RIPEMD160（X1）
func opcodeRipemd160(op *parsedOpcode, vm *Engine) error {
	buf, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}

	vm.dstack.PushByteArray(calcHash(buf, ripemd160.New()))
	return nil
}

//opcodesha1将数据堆栈的顶部项视为原始字节并替换它
//使用sha1（数据）。
//
//堆栈转换：…X1] ->…Sa1（x1）]
func opcodeSha1(op *parsedOpcode, vm *Engine) error {
	buf, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}

	hash := sha1.Sum(buf)
	vm.dstack.PushByteArray(hash[:])
	return nil
}

//opcodesha256将数据堆栈的顶部项视为原始字节并替换
//它带有sha256（数据）。
//
//堆栈转换：…X1] ->…SAH256（X1）
func opcodeSha256(op *parsedOpcode, vm *Engine) error {
	buf, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}

	hash := sha256.Sum256(buf)
	vm.dstack.PushByteArray(hash[:])
	return nil
}

//opcodehash160将数据堆栈的顶部项视为原始字节并替换
//它与ripemd160（sha256（数据））。
//
//堆栈转换：…X1] ->…ripemd160（sha256（x1））]
func opcodeHash160(op *parsedOpcode, vm *Engine) error {
	buf, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}

	hash := sha256.Sum256(buf)
	vm.dstack.PushByteArray(calcHash(hash[:], ripemd160.New()))
	return nil
}

//opcodehash256将数据堆栈的顶部项视为原始字节并替换
//它与sha256（sha256（数据））。
//
//堆栈转换：…X1] ->…sha256（sha256（x1））]
func opcodeHash256(op *parsedOpcode, vm *Engine) error {
	buf, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}

	vm.dstack.PushByteArray(chainhash.DoubleHashB(buf))
	return nil
}

//opcodecodeSeparator将当前脚本偏移量存储为最近的
//请参阅在签名检查期间使用的op_codeseparator。
//
//此操作码不会更改数据堆栈的内容。
func opcodeCodeSeparator(op *parsedOpcode, vm *Engine) error {
	vm.lastCodeSep = vm.scriptOff
	return nil
}

//opcodechecksig将堆栈上的前2个项作为公钥和
//签名并用一个bool替换它们，该bool指示签名是否为
//已成功验证。
//
//验证签名的过程需要在
//与事务签名者的方式相同。它涉及对
//基于哈希类型字节的事务（这是
//签名）以及脚本的最新部分
//操作代码分隔符（或脚本开头，如果没有）
//脚本结束（删除任何其他操作代码分隔符）。一旦这个
//计算“脚本哈希”，使用标准检查签名
//针对提供的公钥的加密方法。
//
//堆栈转换：…签名pubkey]->[…布尔
func opcodeCheckSig(op *parsedOpcode, vm *Engine) error {
	pkBytes, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}

	fullSigBytes, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}

//签名实际上需要比这个长，但是
//下面的哈希类型至少需要1个字节。全长为
//根据脚本标志和分析签名进行检查。
	if len(fullSigBytes) < 1 {
		vm.dstack.PushBool(false)
		return nil
	}

//从签名字符串中删除hashtype并检查
//签名和公钥符合严格的编码要求
//取决于标志。
//
//注意：如果设置了严格编码标志，则
//此处的签名或公共编码会导致即时脚本错误
//（因此不会将结果bool推送到数据堆栈中）。这不同
//从下面的逻辑中分析签名时出现的任何错误
//被视为签名失败，导致错误被推送到
//数据堆栈。这是必需的，因为更一般的脚本
//验证共识规则没有新的严格编码
//由标志启用的需求。
	hashType := SigHashType(fullSigBytes[len(fullSigBytes)-1])
	sigBytes := fullSigBytes[:len(fullSigBytes)-1]
	if err := vm.checkHashTypeEncoding(hashType); err != nil {
		return err
	}
	if err := vm.checkSignatureEncoding(sigBytes); err != nil {
		return err
	}
	if err := vm.checkPubKeyEncoding(pkBytes); err != nil {
		return err
	}

//从最新的操作代码分隔符开始获取脚本。
	subScript := vm.subScript()

//根据签名哈希类型生成签名哈希。
	var hash []byte
	if vm.isWitnessVersionActive(0) {
		var sigHashes *TxSigHashes
		if vm.hashCache != nil {
			sigHashes = vm.hashCache
		} else {
			sigHashes = NewTxSigHashes(&vm.tx)
		}

		hash, err = calcWitnessSignatureHash(subScript, sigHashes, hashType,
			&vm.tx, vm.txIdx, vm.inputAmount)
		if err != nil {
			return err
		}
	} else {
//删除签名，因为没有签名的方法
//自己签字
		subScript = removeOpcodeByData(subScript, fullSigBytes)

		hash = calcSignatureHash(subScript, hashType, &vm.tx, vm.txIdx)
	}

	pubKey, err := btcec.ParsePubKey(pkBytes, btcec.S256())
	if err != nil {
		vm.dstack.PushBool(false)
		return nil
	}

	var signature *btcec.Signature
	if vm.hasFlag(ScriptVerifyStrictEncoding) ||
		vm.hasFlag(ScriptVerifyDERSignatures) {

		signature, err = btcec.ParseDERSignature(sigBytes, btcec.S256())
	} else {
		signature, err = btcec.ParseSignature(sigBytes, btcec.S256())
	}
	if err != nil {
		vm.dstack.PushBool(false)
		return nil
	}

	var valid bool
	if vm.sigCache != nil {
		var sigHash chainhash.Hash
		copy(sigHash[:], hash)

		valid = vm.sigCache.Exists(sigHash, signature, pubKey)
		if !valid && signature.Verify(hash, pubKey) {
			vm.sigCache.Add(sigHash, signature, pubKey)
			valid = true
		}
	} else {
		valid = signature.Verify(hash, pubKey)
	}

	if !valid && vm.hasFlag(ScriptVerifyNullFail) && len(sigBytes) > 0 {
		str := "signature not empty on failed checksig"
		return scriptError(ErrNullFail, str)
	}

	vm.dstack.PushBool(valid)
	return nil
}

//opcodechecksigverify是opcodechecksig和opcodeverify的组合。
//调用opcodechecksig函数，然后调用opcodeverify。见
//有关这些操作码的详细信息，请参阅每个操作码的文档。
//
//堆栈转换：签名pubkey]->[…布尔> >…
func opcodeCheckSigVerify(op *parsedOpcode, vm *Engine) error {
	err := opcodeCheckSig(op, vm)
	if err == nil {
		err = abstractVerify(op, vm, ErrCheckSigVerify)
	}
	return err
}

//ParsedSiginfo包含一个原始签名及其解析形式和一个标志
//是否已经分析过。用于防止解析
//在验证multisig时多次使用相同的签名。
type parsedSigInfo struct {
	signature       []byte
	parsedSignature *btcec.Signature
	parsed          bool
}

//opcodecheckmultisig将堆栈上的顶部项作为整数
//公共密钥，后面跟着许多表示公共的原始数据项
//键，后面跟着整数个签名，后面跟着许多
//作为代表签名的原始数据的条目。
//
//由于原始Satoshi客户机实现中存在错误，因此
//共识规则也要求虚假论点，尽管它不是
//使用。虚拟值应为op_0，尽管该值不是
//共识规则。设置scriptstrictmultisig标志时，它必须
//op0.
//
//上述所有堆栈项都被一个bool替换，bool
//指示是否成功验证了所需的签名数。
//
//请参阅opcodechecksigverify文档以了解有关该过程的更多详细信息
//用于验证每个签名。
//
//堆栈转换：
//…虚拟[sig…]numsigs[pubkey…]numpubkeys]->[…布尔
func opcodeCheckMultiSig(op *parsedOpcode, vm *Engine) error {
	numKeys, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	numPubKeys := int(numKeys.Int32())
	if numPubKeys < 0 {
		str := fmt.Sprintf("number of pubkeys %d is negative",
			numPubKeys)
		return scriptError(ErrInvalidPubKeyCount, str)
	}
	if numPubKeys > MaxPubKeysPerMultiSig {
		str := fmt.Sprintf("too many pubkeys: %d > %d",
			numPubKeys, MaxPubKeysPerMultiSig)
		return scriptError(ErrInvalidPubKeyCount, str)
	}
	vm.numOps += numPubKeys
	if vm.numOps > MaxOpsPerScript {
		str := fmt.Sprintf("exceeded max operation limit of %d",
			MaxOpsPerScript)
		return scriptError(ErrTooManyOperations, str)
	}

	pubKeys := make([][]byte, 0, numPubKeys)
	for i := 0; i < numPubKeys; i++ {
		pubKey, err := vm.dstack.PopByteArray()
		if err != nil {
			return err
		}
		pubKeys = append(pubKeys, pubKey)
	}

	numSigs, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}
	numSignatures := int(numSigs.Int32())
	if numSignatures < 0 {
		str := fmt.Sprintf("number of signatures %d is negative",
			numSignatures)
		return scriptError(ErrInvalidSignatureCount, str)

	}
	if numSignatures > numPubKeys {
		str := fmt.Sprintf("more signatures than pubkeys: %d > %d",
			numSignatures, numPubKeys)
		return scriptError(ErrInvalidSignatureCount, str)
	}

	signatures := make([]*parsedSigInfo, 0, numSignatures)
	for i := 0; i < numSignatures; i++ {
		signature, err := vm.dstack.PopByteArray()
		if err != nil {
			return err
		}
		sigInfo := &parsedSigInfo{signature: signature}
		signatures = append(signatures, sigInfo)
	}

//原始Satoshi客户机实现中的错误意味着还有一个
//必须弹出应使用的堆栈值。不幸的是，这个
//马车的行为现在是共识的一部分，一个硬叉子将是
//需要修复它。
	dummy, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}

//由于不检查伪参数，因此它可以是
//不幸的是，它提供了延展性的来源。因此，
//存在一个脚本标志，用于在值不是0时强制出错。
	if vm.hasFlag(ScriptStrictMultiSig) && len(dummy) != 0 {
		str := fmt.Sprintf("multisig dummy argument has length %d "+
			"instead of 0", len(dummy))
		return scriptError(ErrSigNullDummy, str)
	}

//从最新的操作代码分隔符开始获取脚本。
	script := vm.subScript()

//删除版本0之前的segwit脚本中的签名，因为
//签名本身不可能签名。
	if !vm.isWitnessVersionActive(0) {
		for _, sigInfo := range signatures {
			script = removeOpcodeByData(script, sigInfo.signature)
		}
	}

	success := true
	numPubKeys++
	pubKeyIdx := -1
	signatureIdx := 0
	for numSignatures > 0 {
//当签名多于公钥时，
//因为签名太多，所以无法成功
//无效，请提前退出。
		pubKeyIdx++
		numPubKeys--
		if numSignatures > numPubKeys {
			success = false
			break
		}

		sigInfo := signatures[signatureIdx]
		pubKey := pubKeys[pubKeyIdx]

//签名和公钥评估的顺序是
//这里很重要，因为它可以用
//如果设置了严格编码标志，则不检查multisig。

		rawSig := sigInfo.signature
		if len(rawSig) == 0 {
//如果签名为空，则跳到下一个pubkey。
			continue
		}

//将签名拆分为哈希类型和签名组件。
		hashType := SigHashType(rawSig[len(rawSig)-1])
		signature := rawSig[:len(rawSig)-1]

//只分析和检查一次签名编码。
		var parsedSig *btcec.Signature
		if !sigInfo.parsed {
			if err := vm.checkHashTypeEncoding(hashType); err != nil {
				return err
			}
			if err := vm.checkSignatureEncoding(signature); err != nil {
				return err
			}

//分析签名。
			var err error
			if vm.hasFlag(ScriptVerifyStrictEncoding) ||
				vm.hasFlag(ScriptVerifyDERSignatures) {

				parsedSig, err = btcec.ParseDERSignature(signature,
					btcec.S256())
			} else {
				parsedSig, err = btcec.ParseSignature(signature,
					btcec.S256())
			}
			sigInfo.parsed = true
			if err != nil {
				continue
			}
			sigInfo.parsedSignature = parsedSig
		} else {
//如果签名无效，请跳到下一个pubkey。
			if sigInfo.parsedSignature == nil {
				continue
			}

//使用已分析的签名。
			parsedSig = sigInfo.parsedSignature
		}

		if err := vm.checkPubKeyEncoding(pubKey); err != nil {
			return err
		}

//分析pubkey。
		parsedPubKey, err := btcec.ParsePubKey(pubKey, btcec.S256())
		if err != nil {
			continue
		}

//根据签名哈希类型生成签名哈希。
		var hash []byte
		if vm.isWitnessVersionActive(0) {
			var sigHashes *TxSigHashes
			if vm.hashCache != nil {
				sigHashes = vm.hashCache
			} else {
				sigHashes = NewTxSigHashes(&vm.tx)
			}

			hash, err = calcWitnessSignatureHash(script, sigHashes, hashType,
				&vm.tx, vm.txIdx, vm.inputAmount)
			if err != nil {
				return err
			}
		} else {
			hash = calcSignatureHash(script, hashType, &vm.tx, vm.txIdx)
		}

		var valid bool
		if vm.sigCache != nil {
			var sigHash chainhash.Hash
			copy(sigHash[:], hash)

			valid = vm.sigCache.Exists(sigHash, parsedSig, parsedPubKey)
			if !valid && parsedSig.Verify(hash, parsedPubKey) {
				vm.sigCache.Add(sigHash, parsedSig, parsedPubKey)
				valid = true
			}
		} else {
			valid = parsedSig.Verify(hash, parsedPubKey)
		}

		if valid {
//已验证Pubkey，请转到下一个签名。
			signatureIdx++
			numSignatures--
		}
	}

	if !success && vm.hasFlag(ScriptVerifyNullFail) {
		for _, sig := range signatures {
			if len(sig.signature) > 0 {
				str := "not all signatures empty on failed checkmultisig"
				return scriptError(ErrNullFail, str)
			}
		}
	}

	vm.dstack.PushBool(success)
	return nil
}

//opcodecheckmultisigverify是opcodecheckmultisig和
//OpCo验证。调用opcodecheckmultisig，然后调用opcodeverify。
//有关更多详细信息，请参阅每个操作码的文档。
//
//堆栈转换：
//…虚拟[sig…]numsigs[pubkey…]numpubkeys]->[…布尔> >…
func opcodeCheckMultiSigVerify(op *parsedOpcode, vm *Engine) error {
	err := opcodeCheckMultiSig(op, vm)
	if err == nil {
		err = abstractVerify(op, vm, ErrCheckMultiSigVerify)
	}
	return err
}

//opcodebyname是一个映射，可用于通过其
//人可读名称（op_checkmultisig、op_checksig等）。
var OpcodeByName = make(map[string]byte)

func init() {
//使用
//操作码阵列。同时添加“op_false”、“op_true”和
//“op_nop2”，因为它们是“op_0”、“op_1”的别名，
//分别为“op_checklocktimeverify”和“op_checklocktimeverify”。
	for _, op := range opcodeArray {
		OpcodeByName[op.name] = op.value
	}
	OpcodeByName["OP_FALSE"] = OP_FALSE
	OpcodeByName["OP_TRUE"] = OP_TRUE
	OpcodeByName["OP_NOP2"] = OP_CHECKLOCKTIMEVERIFY
	OpcodeByName["OP_NOP3"] = OP_CHECKSEQUENCEVERIFY
}
