
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//

package blockchain

import (
	"compress/bzip2"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
	_ "github.com/btcsuite/btcd/database/ffldb"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
//testdbtype是用于测试的数据库后端类型。
	testDbType = "ffldb"

//testdbroot是用于创建所有测试数据库的根目录。
	testDbRoot = "testdbs"

//blockdatanet是测试块数据中的预期网络。
	blockDataNet = wire.MainNet
)

//filesexists返回命名文件或目录是否存在。
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

//issupporteddbtype返回传递的数据库类型是否为
//当前支持。
func isSupportedDbType(dbType string) bool {
	supportedDrivers := database.SupportedDrivers()
	for _, driver := range supportedDrivers {
		if dbType == driver {
			return true
		}
	}

	return false
}

//加载块读取包含比特币块数据的文件（gzip，但不包括
//以比特币写入的格式）从磁盘返回
//B切割块。这主要是从BTCDB中的测试代码中借用的。
func loadBlocks(filename string) (blocks []*btcutil.Block, err error) {
	filename = filepath.Join("testdata/", filename)

	var network = wire.MainNet
	var dr io.Reader
	var fi io.ReadCloser

	fi, err = os.Open(filename)
	if err != nil {
		return
	}

	if strings.HasSuffix(filename, ".bz2") {
		dr = bzip2.NewReader(fi)
	} else {
		dr = fi
	}
	defer fi.Close()

	var block *btcutil.Block

	err = nil
	for height := int64(1); err == nil; height++ {
		var rintbuf uint32
		err = binary.Read(dr, binary.LittleEndian, &rintbuf)
		if err == io.EOF {
//以预期偏移量命中文件结尾：无警告
			height--
			err = nil
			break
		}
		if err != nil {
			break
		}
		if rintbuf != uint32(network) {
			break
		}
		err = binary.Read(dr, binary.LittleEndian, &rintbuf)
		blocklen := rintbuf

		rbytes := make([]byte, blocklen)

//读块
		dr.Read(rbytes)

		block, err = btcutil.NewBlockFromBytes(rbytes)
		if err != nil {
			return
		}
		blocks = append(blocks, block)
	}

	return
}

//chainsetup用于创建一个新的数据库和具有genesis的链实例。
//块已插入。除了新的链实例外，它还返回
//调用方在完成清理测试后应调用的拆卸函数。
func chainSetup(dbName string, params *chaincfg.Params) (*BlockChain, func(), error) {
	if !isSupportedDbType(testDbType) {
		return nil, nil, fmt.Errorf("unsupported db type %v", testDbType)
	}

//
//具体处理。
	var db database.DB
	var teardown func()
	if testDbType == "memdb" {
		ndb, err := database.Create(testDbType)
		if err != nil {
			return nil, nil, fmt.Errorf("error creating db: %v", err)
		}
		db = ndb

//设置拆卸功能以进行清理。这个功能是
//返回给调用方，以便在测试完成后调用。
		teardown = func() {
			db.Close()
		}
	} else {
//为测试数据库创建根目录。
		if !fileExists(testDbRoot) {
			if err := os.MkdirAll(testDbRoot, 0700); err != nil {
				err := fmt.Errorf("unable to create test db "+
					"root: %v", err)
				return nil, nil, err
			}
		}

//创建一个新的数据库来存储接受的块。
		dbPath := filepath.Join(testDbRoot, dbName)
		_ = os.RemoveAll(dbPath)
		ndb, err := database.Create(testDbType, dbPath, blockDataNet)
		if err != nil {
			return nil, nil, fmt.Errorf("error creating db: %v", err)
		}
		db = ndb

//设置拆卸功能以进行清理。这个功能是
//返回给调用方，以便在测试完成后调用。
		teardown = func() {
			db.Close()
			os.RemoveAll(dbPath)
			os.RemoveAll(testDbRoot)
		}
	}

//复制链参数以确保测试对其所做的任何修改
//链参数不影响全局实例。
	paramsCopy := *params

//创建主链实例。
	chain, err := New(&Config{
		DB:          db,
		ChainParams: &paramsCopy,
		Checkpoints: nil,
		TimeSource:  NewMedianTime(),
		SigCache:    txscript.NewSigCache(1000),
	})
	if err != nil {
		teardown()
		err := fmt.Errorf("failed to create chain instance: %v", err)
		return nil, nil, err
	}
	return chain, teardown, nil
}

//loadoutxoview返回从文件加载的utxo视图。
func loadUtxoView(filename string) (*UtxoViewpoint, error) {
//utxostore文件格式为：
//<tx hash><output index><serialized utxo len><serialized utxo>
//
//输出索引和序列化的utxo len是小endian uint32s
//序列化的utxo使用chainio.go中描述的格式。

	filename = filepath.Join("testdata", filename)
	fi, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

//根据文件是否压缩，选择“读取”。
	var r io.Reader
	if strings.HasSuffix(filename, ".bz2") {
		r = bzip2.NewReader(fi)
	} else {
		r = fi
	}
	defer fi.Close()

	view := NewUtxoViewpoint()
	for {
//utxo项的哈希。
		var hash chainhash.Hash
		_, err := io.ReadAtLeast(r, hash[:], len(hash[:]))
		if err != nil {
//右偏移处应为EOF。
			if err == io.EOF {
				break
			}
			return nil, err
		}

//utxo项的输出索引。
		var index uint32
		err = binary.Read(r, binary.LittleEndian, &index)
		if err != nil {
			return nil, err
		}

//序列化的utxo条目字节数。
		var numBytes uint32
		err = binary.Read(r, binary.LittleEndian, &numBytes)
		if err != nil {
			return nil, err
		}

//序列化的utxo项。
		serialized := make([]byte, numBytes)
		_, err = io.ReadAtLeast(r, serialized, int(numBytes))
		if err != nil {
			return nil, err
		}

//反序列化并将其添加到视图中。
		entry, err := deserializeUtxoEntry(serialized)
		if err != nil {
			return nil, err
		}
		view.Entries()[wire.OutPoint{Hash: hash, Index: index}] = entry
	}

	return view, nil
}

//convertutxostore从旧格式读取utxostore并将其写回
//使用最新格式。它只对转换utxostore数据有用
//用于已完成的测试。但是，代码是保留的
//可供将来参考。
func convertUtxoStore(r io.Reader, w io.Writer) error {
//旧的utxostore文件格式为：
//<tx hash><serialized utxo len><serialized utxo>
//
//序列化的utxo-len是一个小endian uint32，序列化的
//utxo使用upgrade.go中描述的格式。

	littleEndian := binary.LittleEndian
	for {
//utxo项的哈希。
		var hash chainhash.Hash
		_, err := io.ReadAtLeast(r, hash[:], len(hash[:]))
		if err != nil {
//右偏移处应为EOF。
			if err == io.EOF {
				break
			}
			return err
		}

//序列化的utxo条目字节数。
		var numBytes uint32
		err = binary.Read(r, littleEndian, &numBytes)
		if err != nil {
			return err
		}

//序列化的utxo项。
		serialized := make([]byte, numBytes)
		_, err = io.ReadAtLeast(r, serialized, int(numBytes))
		if err != nil {
			return err
		}

//反序列化条目。
		entries, err := deserializeUtxoEntryV0(serialized)
		if err != nil {
			return err
		}

//循环遍历所有的utxo并在新的
//格式。
		for outputIdx, entry := range entries {
//使用新格式重新序列化条目。
			serialized, err := serializeUtxoEntry(entry)
			if err != nil {
				return err
			}

//
			_, err = w.Write(hash[:])
			if err != nil {
				return err
			}

//
			err = binary.Write(w, littleEndian, outputIdx)
			if err != nil {
				return err
			}

//写入序列化的utxo条目字节数。
			err = binary.Write(w, littleEndian, uint32(len(serialized)))
			if err != nil {
				return err
			}

//编写序列化的utxo。
			_, err = w.Write(serialized)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

//tstsetcoinbasematurity使设置coinbase maturity的功能
//运行测试时可用。
func (b *BlockChain) TstSetCoinbaseMaturity(maturity uint16) {
	b.chainParams.CoinbaseMaturity = maturity
}

//newFakeChain返回一个可用于语法测试的链。它是
//重要的是要注意，此链没有与之关联的数据库，因此
//
//使用它。
func newFakeChain(params *chaincfg.Params) *BlockChain {
//创建一个Genesis块节点并用它填充块索引索引
//用于创建下面的假链。
	node := newBlockNode(&params.GenesisBlock.Header, nil)
	index := newBlockIndex(nil, params)
	index.AddNode(node)

	targetTimespan := int64(params.TargetTimespan / time.Second)
	targetTimePerBlock := int64(params.TargetTimePerBlock / time.Second)
	adjustmentFactor := params.RetargetAdjustmentFactor
	return &BlockChain{
		chainParams:         params,
		timeSource:          NewMedianTime(),
		minRetargetTimespan: targetTimespan / adjustmentFactor,
		maxRetargetTimespan: targetTimespan * adjustmentFactor,
		blocksPerRetarget:   int32(targetTimespan / targetTimePerBlock),
		index:               index,
		bestChain:           newChainView(node),
		warningCaches:       newThresholdCaches(vbNumBits),
		deploymentCaches:    newThresholdCaches(chaincfg.DefinedDeployments),
	}
}

//
//为其他字段提供填充字段和假值。
func newFakeNode(parent *blockNode, blockVersion int32, bits uint32, timestamp time.Time) *blockNode {
//组成一个标题并从中创建一个块节点。
	header := &wire.BlockHeader{
		Version:   blockVersion,
		PrevBlock: parent.hash,
		Bits:      bits,
		Timestamp: timestamp,
	}
	return newBlockNode(header, parent)
}
