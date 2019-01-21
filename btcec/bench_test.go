
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有2013-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package btcec

import "testing"

//BenchmarkAddJacobian将secp256k1曲线AddJacobian函数与
//z值为1，以便使用相关的优化。
func BenchmarkAddJacobian(b *testing.B) {
	b.StopTimer()
	x1 := new(fieldVal).SetHex("34f9460f0e4f08393d192b3c5133a6ba099aa0ad9fd54ebccfacdfa239ff49c6")
	y1 := new(fieldVal).SetHex("0b71ea9bd730fd8923f6d25a7a91e7dd7728a960686cb5a901bb419e0f2ca232")
	z1 := new(fieldVal).SetHex("1")
	x2 := new(fieldVal).SetHex("34f9460f0e4f08393d192b3c5133a6ba099aa0ad9fd54ebccfacdfa239ff49c6")
	y2 := new(fieldVal).SetHex("0b71ea9bd730fd8923f6d25a7a91e7dd7728a960686cb5a901bb419e0f2ca232")
	z2 := new(fieldVal).SetHex("1")
	x3, y3, z3 := new(fieldVal), new(fieldVal), new(fieldVal)
	curve := S256()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		curve.addJacobian(x1, y1, z1, x2, y2, z2, x3, y3, z3)
	}
}

//基准点addjacobiannotzone基准点secp256k1曲线addjacobian
//函数的z值不是1，因此与
//
func BenchmarkAddJacobianNotZOne(b *testing.B) {
	b.StopTimer()
	x1 := new(fieldVal).SetHex("d3e5183c393c20e4f464acf144ce9ae8266a82b67f553af33eb37e88e7fd2718")
	y1 := new(fieldVal).SetHex("5b8f54deb987ec491fb692d3d48f3eebb9454b034365ad480dda0cf079651190")
	z1 := new(fieldVal).SetHex("2")
	x2 := new(fieldVal).SetHex("91abba6a34b7481d922a4bd6a04899d5a686f6cf6da4e66a0cb427fb25c04bd4")
	y2 := new(fieldVal).SetHex("03fede65e30b4e7576a2abefc963ddbf9fdccbf791b77c29beadefe49951f7d1")
	z2 := new(fieldVal).SetHex("3")
	x3, y3, z3 := new(fieldVal), new(fieldVal), new(fieldVal)
	curve := S256()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		curve.addJacobian(x1, y1, z1, x2, y2, z2, x3, y3, z3)
	}
}

//基准scalarbasemult基准secp256k1曲线scalarbasemult
//功能。
func BenchmarkScalarBaseMult(b *testing.B) {
	k := fromHex("d74bf844b0862475103d96a611cf2d898447e288d34b360bc885cb8ce7c00575")
	curve := S256()
	for i := 0; i < b.N; i++ {
		curve.ScalarBaseMult(k.Bytes())
	}
}

//基准scalarbasemultlarge基准secp256k1曲线scalarbasemult
//具有异常大k值的函数。
func BenchmarkScalarBaseMultLarge(b *testing.B) {
	k := fromHex("d74bf844b0862475103d96a611cf2d898447e288d34b360bc885cb8ce7c005751111111011111110")
	curve := S256()
	for i := 0; i < b.N; i++ {
		curve.ScalarBaseMult(k.Bytes())
	}
}

//Benchmarkscalarmult对secp256k1曲线scalarmult函数进行基准测试。
func BenchmarkScalarMult(b *testing.B) {
	x := fromHex("34f9460f0e4f08393d192b3c5133a6ba099aa0ad9fd54ebccfacdfa239ff49c6")
	y := fromHex("0b71ea9bd730fd8923f6d25a7a91e7dd7728a960686cb5a901bb419e0f2ca232")
	k := fromHex("d74bf844b0862475103d96a611cf2d898447e288d34b360bc885cb8ce7c00575")
	curve := S256()
	for i := 0; i < b.N; i++ {
		curve.ScalarMult(x, y, k.Bytes())
	}
}

//基准NAF基准NAF功能。
func BenchmarkNAF(b *testing.B) {
	k := fromHex("d74bf844b0862475103d96a611cf2d898447e288d34b360bc885cb8ce7c00575")
	for i := 0; i < b.N; i++ {
		NAF(k.Bytes())
	}
}

//基准点检验基准点secp256k1曲线
//验证签名。
func BenchmarkSigVerify(b *testing.B) {
	b.StopTimer()
//
//私钥：9e0699c91ca1e3b7e3c9ba71eb71c89890872be97576010fe593ff3fd57e66d
	pubKey := PublicKey{
		Curve: S256(),
		X:     fromHex("d2e670a19c6d753d1a6d8b20bd045df8a08fb162cf508956c31268c6d81ffdab"),
		Y:     fromHex("ab65528eefbb8057aa85d597258a3fbd481a24633bc9b47a9aa045c91371de52"),
	}

//
	msgHash := fromHex("8de472e2399610baaa7f84840547cd409434e31f5d3bd71e4d947f283874f9c0")
	sig := Signature{
		R: fromHex("fef45d2892953aa5bbcdb057b5e98b208f1617a7498af7eb765574e29b5d9c2c"),
		S: fromHex("d47563f52aac6b04b55de236b7c515eb9311757db01e02cff079c3ca6efb063f"),
	}

	if !sig.Verify(msgHash.Bytes(), &pubKey) {
		b.Errorf("Signature failed to verify")
		return
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		sig.Verify(msgHash.Bytes(), &pubKey)
	}
}

//基准场规范化基准点需要多长时间的内部场
//
func BenchmarkFieldNormalize(b *testing.B) {
//Normalize函数为常量时间，因此默认值为fine。
	f := new(fieldVal)
	for i := 0; i < b.N; i++ {
		f.Normalize()
	}
}
