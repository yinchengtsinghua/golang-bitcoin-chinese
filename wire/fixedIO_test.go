
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wire

import (
	"bytes"
	"io"
)

//FixedWriter实现了IO.Writer接口，并有意允许
//通过强制短写来测试错误路径。
type fixedWriter struct {
	b   []byte
	pos int
}

//写入将p的内容写入w。当p的内容将导致
//写入程序超过固定写入程序的最大允许大小，
//返回io.errshortwrite，写入程序保持不变。
//
//这满足IO.Writer接口。
func (w *fixedWriter) Write(p []byte) (n int, err error) {
	lenp := len(p)
	if w.pos+lenp > cap(w.b) {
		return 0, io.ErrShortWrite
	}
	n = lenp
	w.pos += copy(w.b[w.pos:], p)
	return
}

//字节返回已写入固定写入程序的字节。
func (w *fixedWriter) Bytes() []byte {
	return w.b
}

//NewFixedWriter返回一个新的IO.Writer，它将错误超过
//已写入指定的最大值。
func newFixedWriter(max int) io.Writer {
	b := make([]byte, max)
	fw := fixedWriter{b, 0}
	return &fw
}

//FixedReader实现了IO.Reader接口，并有意允许
//通过强制短读来测试错误路径。
type fixedReader struct {
	buf   []byte
	pos   int
	iobuf *bytes.Buffer
}

//读取从固定读卡器读取下一个len（p）字节。当
//读取的字节数将超过允许读取的最大字节数。
//修复了写入程序，返回一个错误。
//
//这满足了IO.reader接口。
func (fr *fixedReader) Read(p []byte) (n int, err error) {
	n, err = fr.iobuf.Read(p)
	fr.pos += n
	return
}

//newFixedReader返回一个新的IO.Reader，该IO.Reader将出错一次以上的字节
//已读取指定的最大值。
func newFixedReader(max int, buf []byte) io.Reader {
	b := make([]byte, max)
	if buf != nil {
		copy(b[:], buf)
	}

	iobuf := bytes.NewBuffer(b)
	fr := fixedReader{b, 0, iobuf}
	return &fr
}
