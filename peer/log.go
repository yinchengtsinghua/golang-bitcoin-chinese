
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package peer

import (
	"fmt"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btclog"
)

const (
//MaxRejectReasonLen是已清除拒绝原因的最大长度
//that will be logged.
	maxRejectReasonLen = 250
)

//日志是一个没有输出过滤器初始化的日志程序。这个
//意味着在调用方之前，包默认不会执行任何日志记录
//请求它。
var log btclog.Logger

//默认的日志记录量为“无”。
func init() {
	DisableLog()
}

//DisableLog禁用所有库日志输出。日志记录输出被禁用
//默认情况下，直到调用uselogger。
func DisableLog() {
	log = btclog.Disabled
}

//uselogger使用指定的记录器输出包日志信息。
func UseLogger(logger btclog.Logger) {
	log = logger
}

//LogClosing是一个可以用%v打印的闭包，用于
//为详细的日志级别创建数据并避免
//数据未打印时的工作。
type logClosure func() string

func (c logClosure) String() string {
	return c()
}

func newLogClosure(c func() string) logClosure {
	return logClosure(c)
}

//DirectionString是一个助手函数，它返回一个表示
//连接的方向（入站或出站）。
func directionString(inbound bool) string {
	if inbound {
		return "inbound"
	}
	return "outbound"
}

//FormatLockTime以可读字符串的形式返回事务锁定时间。
func formatLockTime(lockTime uint32) string {
//事务的锁定时间字段要么是块高度，
//哪个事务已完成或时间戳取决于
//值在LockTimeThreshold之前。当它在
//门槛是一个街区的高度。
	if lockTime < txscript.LockTimeThreshold {
		return fmt.Sprintf("height %d", lockTime)
	}

	return time.Unix(int64(lockTime), 0).String()
}

//invSummary returns an inventory message as a human-readable string.
func invSummary(invList []*wire.InvVect) string {
//没有库存。
	invLen := len(invList)
	if invLen == 0 {
		return "empty"
	}

//一个库存项目。
	if invLen == 1 {
		iv := invList[0]
		switch iv.Type {
		case wire.InvTypeError:
			return fmt.Sprintf("error %s", iv.Hash)
		case wire.InvTypeWitnessBlock:
			return fmt.Sprintf("witness block %s", iv.Hash)
		case wire.InvTypeBlock:
			return fmt.Sprintf("block %s", iv.Hash)
		case wire.InvTypeWitnessTx:
			return fmt.Sprintf("witness tx %s", iv.Hash)
		case wire.InvTypeTx:
			return fmt.Sprintf("tx %s", iv.Hash)
		}

		return fmt.Sprintf("unknown (%d) %s", uint32(iv.Type), iv.Hash)
	}

//多个投资项目。
	return fmt.Sprintf("size %d", invLen)
}

//locatorsummary返回块定位器作为可读字符串。
func locatorSummary(locator []*chainhash.Hash, stopHash *chainhash.Hash) string {
	if len(locator) > 0 {
		return fmt.Sprintf("locator %s, stop %s", locator[0], stopHash)
	}

	return fmt.Sprintf("no locator, stop %s", stopHash)

}

//消毒剂去除任何甚至是非常危险的字符，例如
//作为HTML控制字符，从传递的字符串中提取。它也限制了它。
//传递的最大大小，可以是0表示无限。当字符串是
//受限，它还将向字符串添加“…”以指示它被截断。
func sanitizeString(str string, maxLength uint) string {
	const safeChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXY" +
		"Z01234567890 .,;_/:?@"

//删除不在safechars字符串中的任何字符。
	str = strings.Map(func(r rune) rune {
		if strings.ContainsRune(safeChars, r) {
			return r
		}
		return -1
	}, str)

//将字符串限制为允许的最大长度。
	if maxLength > 0 && uint(len(str)) > maxLength {
		str = str[:maxLength]
		str = str + "..."
	}
	return str
}

//messagesummary返回一个人类可读的字符串，该字符串对消息进行汇总。
//并非所有消息都有或需要摘要。这用于调试日志记录。
func messageSummary(msg wire.Message) string {
	switch msg := msg.(type) {
	case *wire.MsgVersion:
		return fmt.Sprintf("agent %s, pver %d, block %d",
			msg.UserAgent, msg.ProtocolVersion, msg.LastBlock)

	case *wire.MsgVerAck:
//没有摘要。

	case *wire.MsgGetAddr:
//没有摘要。

	case *wire.MsgAddr:
		return fmt.Sprintf("%d addr", len(msg.AddrList))

	case *wire.MsgPing:
//No summary - perhaps add nonce.

	case *wire.MsgPong:
//没有总结-可能添加nonce。

	case *wire.MsgAlert:
//没有摘要。

	case *wire.MsgMemPool:
//没有摘要。

	case *wire.MsgTx:
		return fmt.Sprintf("hash %s, %d inputs, %d outputs, lock %s",
			msg.TxHash(), len(msg.TxIn), len(msg.TxOut),
			formatLockTime(msg.LockTime))

	case *wire.MsgBlock:
		header := &msg.Header
		return fmt.Sprintf("hash %s, ver %d, %d tx, %s", msg.BlockHash(),
			header.Version, len(msg.Transactions), header.Timestamp)

	case *wire.MsgInv:
		return invSummary(msg.InvList)

	case *wire.MsgNotFound:
		return invSummary(msg.InvList)

	case *wire.MsgGetData:
		return invSummary(msg.InvList)

	case *wire.MsgGetBlocks:
		return locatorSummary(msg.BlockLocatorHashes, &msg.HashStop)

	case *wire.MsgGetHeaders:
		return locatorSummary(msg.BlockLocatorHashes, &msg.HashStop)

	case *wire.MsgHeaders:
		return fmt.Sprintf("num %d", len(msg.Headers))

	case *wire.MsgGetCFHeaders:
		return fmt.Sprintf("start_height=%d, stop_hash=%v",
			msg.StartHeight, msg.StopHash)

	case *wire.MsgCFHeaders:
		return fmt.Sprintf("stop_hash=%v, num_filter_hashes=%d",
			msg.StopHash, len(msg.FilterHashes))

	case *wire.MsgReject:
//确保可变长度字符串不包含任何
//甚至是极其危险的字符，如HTML
//控制字符等也限制它们的正常长度
//登录中。
		rejCommand := sanitizeString(msg.Cmd, wire.CommandSize)
		rejReason := sanitizeString(msg.Reason, maxRejectReasonLen)
		summary := fmt.Sprintf("cmd %v, code %v, reason %v", rejCommand,
			msg.Code, rejReason)
		if rejCommand == wire.CmdBlock || rejCommand == wire.CmdTx {
			summary += fmt.Sprintf(", hash %v", msg.Hash)
		}
		return summary
	}

//其他邮件没有摘要。
	return ""
}
