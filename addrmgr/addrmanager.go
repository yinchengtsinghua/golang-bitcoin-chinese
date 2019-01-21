
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2016 BTCSuite开发者
//版权所有（c）2015-2018法令开发商
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package addrmgr

import (
	"container/list"
crand "crypto/rand" //播种
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

//addrmanager提供了一个并发安全地址管理器来缓存潜在的
//比特币网络上的对等方。
type AddrManager struct {
	mtx            sync.Mutex
	peersFile      string
	lookupFunc     func(string) ([]net.IP, error)
	rand           *rand.Rand
	key            [32]byte
addrIndex      map[string]*KnownAddress //所有加法器的KA地址键。
	addrNew        [newBucketCount]map[string]*KnownAddress
	addrTried      [triedBucketCount]*list.List
	started        int32
	shutdown       int32
	wg             sync.WaitGroup
	quit           chan struct{}
	nTried         int
	nNew           int
	lamtx          sync.Mutex
	localAddresses map[string]*localAddress
}

type serializedKnownAddress struct {
	Addr        string
	Src         string
	Attempts    int
	TimeStamp   int64
	LastAttempt int64
	LastSuccess int64
//上下文中没有可用的refcount或tryed。
}

type serializedAddrManager struct {
	Version      int
	Key          [32]byte
	Addresses    []*serializedKnownAddress
NewBuckets   [newBucketCount][]string //字符串是netaddresskey
	TriedBuckets [triedBucketCount][]string
}

type localAddress struct {
	na    *wire.NetAddress
	score AddressPriority
}

//AddressPriority类型用于描述本地地址的层次结构
//发现方法。
type AddressPriority int

const (
//interfaceprio表示地址在本地接口上
	InterfacePrio AddressPriority = iota

//BOUNDPRIO表示地址已明确绑定到。
	BoundPrio

//upnpprio表示地址是从upnp获得的。
	UpnpPrio

//
	HTTPPrio

//manualprio表示地址由--externalIP提供。
	ManualPrio
)

const (
//NeedAddressThreshold是在其中
//地址管理器将声称需要更多地址。
	needAddressThreshold = 1000

//dumpAddressInterval是用于转储地址的间隔
//缓存到磁盘以备将来使用。
	dumpAddressInterval = time.Minute * 10

//TriedBucketSize是每个地址中的最大地址数。
//尝试了地址存储桶。
	triedBucketSize = 256

//TriedBucketCount是我们尝试拆分的存储桶数。
//地址结束。
	triedBucketCount = 64

//NewBucketSize是每个新地址中的最大地址数。
//桶。
	newBucketSize = 64

//NewBucketCount是我们分配新地址的存储桶数。
//结束。
	newBucketCount = 1024

//TriedBucketsPerGroup是在其上
//地址组将分散。
	triedBucketsPerGroup = 8

//NewBucketsPerGroup是在其上
//源地址组将分散。
	newBucketsPerGroup = 64

//NewBucketsPerAddress是常见的新存储桶数
//地址可能会结束。
	newBucketsPerAddress = 8

//nummissingdays是我们假定
//如果我们没有看到它在那么长时间内宣布的话，地址就消失了。
	numMissingDays = 30

//NumRetries是以前没有一次成功的尝试次数
//我们假设地址不好。
	numRetries = 3

//MaxFailures是我们接受的最大失败数
//在考虑地址错误之前取得成功。
	maxFailures = 10

//minbaddays是自上次成功以来的天数
//将考虑逐出地址。
	minBadDays = 7

//getaddrmax是我们将在响应中发送的最多地址
//到getaddr（在实践中，我们将从
//调用AddressCache（））。
	getAddrMax = 2500

//GetAddrPercent是已知的
//将与对AddressCache的调用共享。
	getAddrPercent = 23

//serialisationversion是磁盘格式的当前版本。
	serialisationVersion = 1
)

//updateAddress是更新已知地址的帮助函数
//到地址管理器，或者添加地址（如果还不知道）。
func (a *AddrManager) updateAddress(netAddr, srcAddr *wire.NetAddress) {
//筛选出不可路由的地址。注意，不可路由
//还包括无效和本地地址。
	if !IsRoutable(netAddr) {
		return
	}

	addr := NetAddressKey(netAddr)
	ka := a.find(netAddr)
	if ka != nil {
//TODO:仅定期更新地址。
//更新上次看到的时间和服务。
//注意，为了防止在getaddr上产生过多的垃圾
//消息AddrManger中的网络地址是*不可变*，
//如果我们需要更改它们，则将指针替换为
//新副本，这样我们就不必为getaddr复制每个NA。
		if netAddr.Timestamp.After(ka.na.Timestamp) ||
			(ka.na.Services&netAddr.Services) !=
				netAddr.Services {

			naCopy := *ka.na
			naCopy.Timestamp = netAddr.Timestamp
			naCopy.AddService(netAddr.Services)
			ka.na = &naCopy
		}

//如果已经试过了，我们就没什么可做的了。
		if ka.tried {
			return
		}

//已经达到极限了？
		if ka.refs == newBucketsPerAddress {
			return
		}

//我们拥有的条目越多，我们添加的可能性就越小。
//可能性为2n。
		factor := int32(2 * ka.refs)
		if a.rand.Int31n(factor) != 0 {
			return
		}
	} else {
//复制网络地址以避免竞争，因为它是
//在addrmanager代码的其他地方更新，否则
//更改对等机上的实际网络地址。
		netAddrCopy := *netAddr
		ka = &KnownAddress{na: &netAddrCopy, srcAddr: srcAddr}
		a.addrIndex[addr] = ka
		a.nNew++
//XXX时间惩罚？
	}

	bucket := a.getNewBucket(netAddr, srcAddr)

//已经存在？
	if _, ok := a.addrNew[bucket][addr]; ok {
		return
	}

//强制最大地址。
	if len(a.addrNew[bucket]) > newBucketSize {
		log.Tracef("new bucket is full, expiring old")
		a.expireNew(bucket)
	}

//添加到新存储桶。
	ka.refs++
	a.addrNew[bucket][addr] = ka

	log.Tracef("Added new address %s for a total of %d addresses", addr,
		a.nTried+a.nNew)
}

//expirenew通过使真正坏的条目过期来在新存储桶中腾出空间。
//如果没有坏的条目可用，我们会查看一些并删除最旧的条目。
func (a *AddrManager) expireNew(bucket int) {
//首先看看是否有任何条目是如此糟糕，我们可以扔
//他们离开了。否则，我们会丢弃缓存中最旧的条目。
//比特币在这里选择四个随机的，只扔最老的
//但我们在最初的遍历中跟踪最旧的
//用这些信息来代替。
	var oldest *KnownAddress
	for k, v := range a.addrNew[bucket] {
		if v.isBad() {
			log.Tracef("expiring bad address %v", k)
			delete(a.addrNew[bucket], k)
			v.refs--
			if v.refs == 0 {
				a.nNew--
				delete(a.addrIndex, k)
			}
			continue
		}
		if oldest == nil {
			oldest = v
		} else if !v.na.Timestamp.After(oldest.na.Timestamp) {
			oldest = v
		}
	}

	if oldest != nil {
		key := NetAddressKey(oldest.na)
		log.Tracef("expiring oldest address %v", key)

		delete(a.addrNew[bucket], key)
		oldest.refs--
		if oldest.refs == 0 {
			a.nNew--
			delete(a.addrIndex, key)
		}
	}
}

//PickTried从尝试的存储桶中选择要收回的地址。
//我们只选最年长的。比特币选择4个随机条目并丢弃
//他们中的老年人。
func (a *AddrManager) pickTried(bucket int) *list.Element {
	var oldest *KnownAddress
	var oldestElem *list.Element
	for e := a.addrTried[bucket].Front(); e != nil; e = e.Next() {
		ka := e.Value.(*KnownAddress)
		if oldest == nil || oldest.na.Timestamp.After(ka.na.Timestamp) {
			oldestElem = e
			oldest = ka
		}

	}
	return oldestElem
}

func (a *AddrManager) getNewBucket(netAddr, srcAddr *wire.NetAddress) int {
//bitcoind：
//doublesha256（key+source group+int64（doublesha256（key+group+source group））%bucket_per_source_group）%num_new_bucket

	data1 := []byte{}
	data1 = append(data1, a.key[:]...)
	data1 = append(data1, []byte(GroupKey(netAddr))...)
	data1 = append(data1, []byte(GroupKey(srcAddr))...)
	hash1 := chainhash.DoubleHashB(data1)
	hash64 := binary.LittleEndian.Uint64(hash1)
	hash64 %= newBucketsPerGroup
	var hashbuf [8]byte
	binary.LittleEndian.PutUint64(hashbuf[:], hash64)
	data2 := []byte{}
	data2 = append(data2, a.key[:]...)
	data2 = append(data2, GroupKey(srcAddr)...)
	data2 = append(data2, hashbuf[:]...)

	hash2 := chainhash.DoubleHashB(data2)
	return int(binary.LittleEndian.Uint64(hash2) % newBucketCount)
}

func (a *AddrManager) getTriedBucket(netAddr *wire.NetAddress) int {
//比特币将此列为：
//doublesha256（key+group+truncate_to64bits（doublesha256（key））%buckets_per_group）%num_buckets
	data1 := []byte{}
	data1 = append(data1, a.key[:]...)
	data1 = append(data1, []byte(NetAddressKey(netAddr))...)
	hash1 := chainhash.DoubleHashB(data1)
	hash64 := binary.LittleEndian.Uint64(hash1)
	hash64 %= triedBucketsPerGroup
	var hashbuf [8]byte
	binary.LittleEndian.PutUint64(hashbuf[:], hash64)
	data2 := []byte{}
	data2 = append(data2, a.key[:]...)
	data2 = append(data2, GroupKey(netAddr)...)
	data2 = append(data2, hashbuf[:]...)

	hash2 := chainhash.DoubleHashB(data2)
	return int(binary.LittleEndian.Uint64(hash2) % triedBucketCount)
}

//AddressHandler是地址管理器的主要处理程序。必须运行
//作为一个傀儡。
func (a *AddrManager) addressHandler() {
	dumpAddressTicker := time.NewTicker(dumpAddressInterval)
	defer dumpAddressTicker.Stop()
out:
	for {
		select {
		case <-dumpAddressTicker.C:
			a.savePeers()

		case <-a.quit:
			break out
		}
	}
	a.savePeers()
	a.wg.Done()
	log.Trace("Address handler done")
}

//savepeers将所有已知地址保存到一个文件中，以便可以读取这些地址。
//下一次跑步。
func (a *AddrManager) savePeers() {
	a.mtx.Lock()
	defer a.mtx.Unlock()

//首先，我们创建一个可序列化的数据结构，以便将其编码为
//杰森。
	sam := new(serializedAddrManager)
	sam.Version = serialisationVersion
	copy(sam.Key[:], a.key[:])

	sam.Addresses = make([]*serializedKnownAddress, len(a.addrIndex))
	i := 0
	for k, v := range a.addrIndex {
		ska := new(serializedKnownAddress)
		ska.Addr = k
		ska.TimeStamp = v.na.Timestamp.Unix()
		ska.Src = NetAddressKey(v.srcAddr)
		ska.Attempts = v.attempts
		ska.LastAttempt = v.lastattempt.Unix()
		ska.LastSuccess = v.lastsuccess.Unix()
//尝试和引用在结构的其余部分中是隐式的
//并将从非宗教化的背景中进行计算。
		sam.Addresses[i] = ska
		i++
	}
	for i := range a.addrNew {
		sam.NewBuckets[i] = make([]string, len(a.addrNew[i]))
		j := 0
		for k := range a.addrNew[i] {
			sam.NewBuckets[i][j] = k
			j++
		}
	}
	for i := range a.addrTried {
		sam.TriedBuckets[i] = make([]string, a.addrTried[i].Len())
		j := 0
		for e := a.addrTried[i].Front(); e != nil; e = e.Next() {
			ka := e.Value.(*KnownAddress)
			sam.TriedBuckets[i][j] = NetAddressKey(ka.na)
			j++
		}
	}

	w, err := os.Create(a.peersFile)
	if err != nil {
		log.Errorf("Error opening file %s: %v", a.peersFile, err)
		return
	}
	enc := json.NewEncoder(w)
	defer w.Close()
	if err := enc.Encode(&sam); err != nil {
		log.Errorf("Failed to encode file %s: %v", a.peersFile, err)
		return
	}
}

//loadpeers从保存的文件中加载已知地址。如果为空、丢失或
//格式不正确的文件，只需不加载任何内容并重新开始
func (a *AddrManager) loadPeers() {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	err := a.deserializePeers(a.peersFile)
	if err != nil {
		log.Errorf("Failed to parse file %s: %v", a.peersFile, err)
//如果它是无效的，我们就无条件地核老的。
		err = os.Remove(a.peersFile)
		if err != nil {
			log.Warnf("Failed to remove corrupt peers file %s: %v",
				a.peersFile, err)
		}
		a.reset()
		return
	}
	log.Infof("Loaded %d addresses from file '%s'", a.numAddresses(), a.peersFile)
}

func (a *AddrManager) deserializePeers(filePath string) error {

	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil
	}
	r, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("%s error opening file: %v", filePath, err)
	}
	defer r.Close()

	var sam serializedAddrManager
	dec := json.NewDecoder(r)
	err = dec.Decode(&sam)
	if err != nil {
		return fmt.Errorf("error reading %s: %v", filePath, err)
	}

	if sam.Version != serialisationVersion {
		return fmt.Errorf("unknown version %v in serialized "+
			"addrmanager", sam.Version)
	}
	copy(a.key[:], sam.Key[:])

	for _, v := range sam.Addresses {
		ka := new(KnownAddress)
		ka.na, err = a.DeserializeNetAddress(v.Addr)
		if err != nil {
			return fmt.Errorf("failed to deserialize netaddress "+
				"%s: %v", v.Addr, err)
		}
		ka.srcAddr, err = a.DeserializeNetAddress(v.Src)
		if err != nil {
			return fmt.Errorf("failed to deserialize netaddress "+
				"%s: %v", v.Src, err)
		}
		ka.attempts = v.Attempts
		ka.lastattempt = time.Unix(v.LastAttempt, 0)
		ka.lastsuccess = time.Unix(v.LastSuccess, 0)
		a.addrIndex[NetAddressKey(ka.na)] = ka
	}

	for i := range sam.NewBuckets {
		for _, val := range sam.NewBuckets[i] {
			ka, ok := a.addrIndex[val]
			if !ok {
				return fmt.Errorf("newbucket contains %s but "+
					"none in address list", val)
			}

			if ka.refs == 0 {
				a.nNew++
			}
			ka.refs++
			a.addrNew[i][val] = ka
		}
	}
	for i := range sam.TriedBuckets {
		for _, val := range sam.TriedBuckets[i] {
			ka, ok := a.addrIndex[val]
			if !ok {
				return fmt.Errorf("Newbucket contains %s but "+
					"none in address list", val)
			}

			ka.tried = true
			a.nTried++
			a.addrTried[i].PushBack(ka)
		}
	}

//健康检查。
	for k, v := range a.addrIndex {
		if v.refs == 0 && !v.tried {
			return fmt.Errorf("address %s after serialisation "+
				"with no references", k)
		}

		if v.refs > 0 && v.tried {
			return fmt.Errorf("address %s after serialisation "+
				"which is both new and tried!", k)
		}
	}

	return nil
}

//DeserializeNetAddress将给定的地址字符串转换为*Wire.NetAddress
func (a *AddrManager) DeserializeNetAddress(addr string) (*wire.NetAddress, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, err
	}

	return a.HostToNetAddress(host, uint16(port), wire.SFNodeNetwork)
}

//Start开始管理已知池的核心地址处理程序
//地址、超时和基于间隔的写入。
func (a *AddrManager) Start() {
//已经开始？
	if atomic.AddInt32(&a.started, 1) != 1 {
		return
	}

	log.Trace("Starting address manager")

//从文件中加载我们已经知道的对等点。
	a.loadPeers()

//启动地址标记器定期保存地址。
	a.wg.Add(1)
	go a.addressHandler()
}

//stop通过停止主处理程序优雅地关闭地址管理器。
func (a *AddrManager) Stop() error {
	if atomic.AddInt32(&a.shutdown, 1) != 1 {
		log.Warnf("Address manager is already in the process of " +
			"shutting down")
		return nil
	}

	log.Infof("Address manager shutting down")
	close(a.quit)
	a.wg.Wait()
	return nil
}

//addaddresses向地址管理器添加新地址。它强制执行最大值
//地址的数目，并且无提示地忽略重复的地址。它是
//同时访问安全。
func (a *AddrManager) AddAddresses(addrs []*wire.NetAddress, srcAddr *wire.NetAddress) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	for _, na := range addrs {
		a.updateAddress(na, srcAddr)
	}
}

//addaddress向地址管理器添加新地址。它强制执行最大值
//地址的数目，并且无提示地忽略重复的地址。它是
//同时访问安全。
func (a *AddrManager) AddAddress(addr, srcAddr *wire.NetAddress) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	a.updateAddress(addr, srcAddr)
}

//addAddressByIP添加了一个地址，其中我们被赋予了一个ip:port而不是
//网络地址。
func (a *AddrManager) AddAddressByIP(addrIP string) error {
//拆分IP和端口
	addr, portStr, err := net.SplitHostPort(addrIP)
	if err != nil {
		return err
	}
//把它放到网上。网址
	ip := net.ParseIP(addr)
	if ip == nil {
		return fmt.Errorf("invalid ip address %s", addr)
	}
	port, err := strconv.ParseUint(portStr, 10, 0)
	if err != nil {
		return fmt.Errorf("invalid port %s: %v", portStr, err)
	}
	na := wire.NewNetAddressIPPort(ip, uint16(port), 0)
a.AddAddress(na, na) //使用正确的SRC地址
	return nil
}

//numAddresses返回地址管理器已知的地址数。
func (a *AddrManager) numAddresses() int {
	return a.nTried + a.nNew
}

//numAddresses返回地址管理器已知的地址数。
func (a *AddrManager) NumAddresses() int {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	return a.numAddresses()
}

//NeedMoreAddresses返回地址管理器是否需要更多
//地址。
func (a *AddrManager) NeedMoreAddresses() bool {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	return a.numAddresses() < needAddressThreshold
}

//AddressCache返回当前地址缓存。它必须被视为
//只读（但由于它现在是副本，所以这并不危险）。
func (a *AddrManager) AddressCache() []*wire.NetAddress {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	addrIndexLen := len(a.addrIndex)
	if addrIndexLen == 0 {
		return nil
	}

	allAddr := make([]*wire.NetAddress, 0, addrIndexLen)
//迭代顺序在这里是未定义的，但我们还是随机化了它。
	for _, v := range a.addrIndex {
		allAddr = append(allAddr, v.na)
	}

	numAddresses := addrIndexLen * getAddrPercent / 100
	if numAddresses > getAddrMax {
		numAddresses = getAddrMax
	}

//费希尔·耶茨洗牌。我们只需要先做
//“numAddresses”，因为我们正在丢弃其余的。
	for i := 0; i < numAddresses; i++ {
//在当前索引和结尾之间选取一个数字
		j := rand.Intn(addrIndexLen-i) + i
		allAddr[i], allAddr[j] = allAddr[j], allAddr[i]
	}

//削减我们愿意分享的限制。
	return allAddr[0:numAddresses]
}

//重置通过重新初始化随机源来重置地址管理器
//分配新的空桶存储。
func (a *AddrManager) reset() {

	a.addrIndex = make(map[string]*KnownAddress)

//用来自好的随机源的字节填充键。
	io.ReadFull(crand.Reader, a.key[:])
	for i := range a.addrNew {
		a.addrNew[i] = make(map[string]*KnownAddress)
	}
	for i := range a.addrTried {
		a.addrTried[i] = list.New()
	}
}

//hostToNetAddress返回给定主机地址的网络地址。如果地址
//是一个Tor。洋葱地址将被处理。如果主机是
//不是IP地址，它将被解析（如果需要，通过Tor）。
func (a *AddrManager) HostToNetAddress(host string, port uint16, services wire.ServiceFlag) (*wire.NetAddress, error) {
//Tor地址是16个字符base32+“.onion”
	var ip net.IP
	if len(host) == 22 && host[16:] == ".onion" {
//go base32编码使用大写字母（与RFC一样
//但是Tor和Bitcoin倾向于使用小写，所以我们切换
//这里是案例。
		data, err := base32.StdEncoding.DecodeString(
			strings.ToUpper(host[:16]))
		if err != nil {
			return nil, err
		}
		prefix := []byte{0xfd, 0x87, 0xd8, 0x7e, 0xeb, 0x43}
		ip = net.IP(append(prefix, data...))
	} else if ip = net.ParseIP(host); ip == nil {
		ips, err := a.lookupFunc(host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("no addresses found for %s", host)
		}
		ip = ips[0]
	}

	return wire.NewNetAddressIPPort(ip, port, services), nil
}

//ip string从提供的网络地址返回IP的字符串。如果
//IP在用于Tor地址的范围内，然后将其转换为
//相关的洋葱地址。
func ipString(na *wire.NetAddress) string {
	if IsOnionCatTor(na) {
//我们现在知道na ip足够长了。
		base32 := base32.StdEncoding.EncodeToString(na.IP[6:])
		return strings.ToLower(base32) + ".onion"
	}

	return na.IP.String()
}

//netaddresskey返回ipv4地址的ip:port形式的字符串密钥
//或[IP]：IPv6地址的端口。
func NetAddressKey(na *wire.NetAddress) string {
	port := strconv.FormatUint(uint64(na.Port), 10)

	return net.JoinHostPort(ipString(na), port)
}

//GetAddress返回一个可路由的地址。它选择了一个
//从可能的地址中随机选择一个，优先选择那些
//最近未使用，不应选择“关闭”地址
//连续地。
func (a *AddrManager) GetAddress() *KnownAddress {
//保护并发访问。
	a.mtx.Lock()
	defer a.mtx.Unlock()

	if a.numAddresses() == 0 {
		return nil
	}

//使用50%的机会在尝试过的表条目和新表条目之间进行选择。
	if a.nTried > 0 && (a.nNew == 0 || a.rand.Intn(2) == 0) {
//尝试进入。
		large := 1 << 30
		factor := 1.0
		for {
//随便挑一个桶。
			bucket := a.rand.Intn(len(a.addrTried))
			if a.addrTried[bucket].Len() == 0 {
				continue
			}

//在列表中选择一个随机条目
			e := a.addrTried[bucket].Front()
			for i :=
				a.rand.Int63n(int64(a.addrTried[bucket].Len())); i > 0; i-- {
				e = e.Next()
			}
			ka := e.Value.(*KnownAddress)
			randval := a.rand.Intn(large)
			if float64(randval) < (factor * ka.chance() * float64(large)) {
				log.Tracef("Selected %v from tried bucket",
					NetAddressKey(ka.na))
				return ka
			}
			factor *= 1.2
		}
	} else {
//新节点。
//XXX使用一个闭包/函数来避免重复这个过程。
		large := 1 << 30
		factor := 1.0
		for {
//随便挑一个桶。
			bucket := a.rand.Intn(len(a.addrNew))
			if len(a.addrNew[bucket]) == 0 {
				continue
			}
//然后，随机输入。
			var ka *KnownAddress
			nth := a.rand.Intn(len(a.addrNew[bucket]))
			for _, value := range a.addrNew[bucket] {
				if nth == 0 {
					ka = value
				}
				nth--
			}
			randval := a.rand.Intn(large)
			if float64(randval) < (factor * ka.chance() * float64(large)) {
				log.Tracef("Selected %v from new bucket",
					NetAddressKey(ka.na))
				return ka
			}
			factor *= 1.2
		}
	}
}

func (a *AddrManager) find(addr *wire.NetAddress) *KnownAddress {
	return a.addrIndex[NetAddressKey(addr)]
}

//尝试增加给定地址的尝试计数器和更新
//上次尝试时间。
func (a *AddrManager) Attempt(addr *wire.NetAddress) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

//查找地址。
//地址现在肯定会被试用吗？
	ka := a.find(addr)
	if ka == nil {
		return
	}
//将上次尝试时间设置为现在
	ka.attempts++
	ka.lastattempt = time.Now()
}

//Connected将给定地址标记为当前已连接，并在
//当前时间。地址必须已经为addrmanager所知道，否则它将
//被忽视。
func (a *AddrManager) Connected(addr *wire.NetAddress) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	ka := a.find(addr)
	if ka == nil {
		return
	}

//更新时间，从上次更新到现在已经20分钟了
//所以。
	now := time.Now()
	if now.After(ka.na.Timestamp.Add(time.Minute * 20)) {
//KA.NA是不变的，所以更换它。
		naCopy := *ka.na
		naCopy.Timestamp = time.Now()
		ka.na = &naCopy
	}
}

//好的标记给的地址是好的。在一个成功的
//连接和版本交换。如果地址未知
//管理器它将被忽略。
func (a *AddrManager) Good(addr *wire.NetAddress) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	ka := a.find(addr)
	if ka == nil {
		return
	}

//ka.此处不更新时间戳以避免信息泄漏
//关于当前连接的对等机。
	now := time.Now()
	ka.lastsuccess = now
	ka.lastattempt = now
	ka.attempts = 0

//移动到“已尝试设置”，如果需要，也可以删除其他地址。
	if ka.tried {
		return
	}

//好的，需要把它移到试一下。

//从所有新铲斗上拆下。
//记录一个有问题的桶，称之为“第一个”
	addrKey := NetAddressKey(addr)
	oldBucket := -1
	for i := range a.addrNew {
//我们检查是否存在以便记录第一个
		if _, ok := a.addrNew[i][addrKey]; ok {
			delete(a.addrNew[i], addrKey)
			ka.refs--
			if oldBucket == -1 {
				oldBucket = i
			}
		}
	}
	a.nNew--

	if oldBucket == -1 {
//什么？根本不在桶里…恐慌？
		return
	}

	bucket := a.getTriedBucket(ka.na)

//在这个试过的桶里有房间吗？
	if a.addrTried[bucket].Len() < triedBucketSize {
		ka.tried = true
		a.addrTried[bucket].PushBack(ka)
		a.nTried++
		return
	}

//没有空间，我们得把别的东西赶出去。
	entry := a.pickTried(bucket)
	rmka := entry.Value.(*KnownAddress)

//第一个桶应该放进去了。
	newBucket := a.getNewBucket(rmka.na, rmka.srcAddr)

//如果原来的桶里没有空间，我们就把它放进桶里
//释放了一个空间。
	if len(a.addrNew[newBucket]) >= newBucketSize {
		newBucket = oldBucket
	}

//替换为列表中的ka。
	ka.tried = true
	entry.Value = ka

	rmka.tried = false
	rmka.refs++

//我们不碰这里的中心，因为试过的次数保持不变。
//但我们在上面减了一个新的，再提高一次
//回来了。
	a.nNew++

	rmkey := NetAddressKey(rmka.na)
	log.Tracef("Replacing %s with %s in tried", rmkey, addrKey)

//我们确保上面有空间。
	a.addrNew[newBucket][rmkey] = rmka
}

//setServices将given地址的服务设置为提供的值。
func (a *AddrManager) SetServices(addr *wire.NetAddress, services wire.ServiceFlag) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	ka := a.find(addr)
	if ka == nil {
		return
	}

//如果需要，请更新服务。
	if ka.na.Services != services {
//KA.NA是不变的，所以更换它。
		naCopy := *ka.na
		naCopy.Services = services
		ka.na = &naCopy
	}
}

//addlocaladdress将na添加到要公布的已知本地地址列表中
//优先考虑。
func (a *AddrManager) AddLocalAddress(na *wire.NetAddress, priority AddressPriority) error {
	if !IsRoutable(na) {
		return fmt.Errorf("address %s is not routable", na.IP)
	}

	a.lamtx.Lock()
	defer a.lamtx.Unlock()

	key := NetAddressKey(na)
	la, ok := a.localAddresses[key]
	if !ok || la.score < priority {
		if ok {
			la.score = priority + 1
		} else {
			a.localAddresses[key] = &localAddress{
				na:    na,
				score: priority,
			}
		}
	}
	return nil
}

//GetReachabilityFrom返回提供的本地
//提供的远程地址的地址。
func getReachabilityFrom(localAddr, remoteAddr *wire.NetAddress) int {
	const (
		Unreachable = 0
		Default     = iota
		Teredo
		Ipv6Weak
		Ipv4
		Ipv6Strong
		Private
	)

	if !IsRoutable(remoteAddr) {
		return Unreachable
	}

	if IsOnionCatTor(remoteAddr) {
		if IsOnionCatTor(localAddr) {
			return Private
		}

		if IsRoutable(localAddr) && IsIPv4(localAddr) {
			return Ipv4
		}

		return Default
	}

	if IsRFC4380(remoteAddr) {
		if !IsRoutable(localAddr) {
			return Default
		}

		if IsRFC4380(localAddr) {
			return Teredo
		}

		if IsIPv4(localAddr) {
			return Ipv4
		}

		return Ipv6Weak
	}

	if IsIPv4(remoteAddr) {
		if IsRoutable(localAddr) && IsIPv4(localAddr) {
			return Ipv4
		}
		return Unreachable
	}

 /*IPv6*/
	var tunnelled bool
//我们的V6是隧道式的吗？
	if IsRFC3964(localAddr) || IsRFC6052(localAddr) || IsRFC6145(localAddr) {
		tunnelled = true
	}

	if !IsRoutable(localAddr) {
		return Default
	}

	if IsRFC4380(localAddr) {
		return Teredo
	}

	if IsIPv4(localAddr) {
		return Ipv4
	}

	if tunnelled {
//只有当我们不挖掘IPv6时才优先考虑它。
		return Ipv6Weak
	}

	return Ipv6Strong
}

//GetBestLocalAddress返回要使用的最合适的本地地址
//对于给定的远程地址。
func (a *AddrManager) GetBestLocalAddress(remoteAddr *wire.NetAddress) *wire.NetAddress {
	a.lamtx.Lock()
	defer a.lamtx.Unlock()

	bestreach := 0
	var bestscore AddressPriority
	var bestAddress *wire.NetAddress
	for _, la := range a.localAddresses {
		reach := getReachabilityFrom(la.na, remoteAddr)
		if reach > bestreach ||
			(reach == bestreach && la.score > bestscore) {
			bestreach = reach
			bestscore = la.score
			bestAddress = la.na
		}
	}
	if bestAddress != nil {
		log.Debugf("Suggesting address %s:%d for %s:%d", bestAddress.IP,
			bestAddress.Port, remoteAddr.IP, remoteAddr.Port)
	} else {
		log.Debugf("No worthy address for %s:%d", remoteAddr.IP,
			remoteAddr.Port)

//如果没有合适的话，送一些无法用的东西。
		var ip net.IP
		if !IsIPv4(remoteAddr) && !IsOnionCatTor(remoteAddr) {
			ip = net.IPv6zero
		} else {
			ip = net.IPv4zero
		}
		services := wire.SFNodeNetwork | wire.SFNodeWitness | wire.SFNodeBloom
		bestAddress = wire.NewNetAddressIPPort(ip, 0, services)
	}

	return bestAddress
}

//new返回新的比特币地址管理器。
//使用Start开始处理异步地址更新。
func New(dataDir string, lookupFunc func(string) ([]net.IP, error)) *AddrManager {
	am := AddrManager{
		peersFile:      filepath.Join(dataDir, "peers.json"),
		lookupFunc:     lookupFunc,
		rand:           rand.New(rand.NewSource(time.Now().UnixNano())),
		quit:           make(chan struct{}),
		localAddresses: make(map[string]*localAddress),
	}
	am.reset()
	return &am
}
