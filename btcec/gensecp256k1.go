
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

//
//
//

package btcec

//参考文献：
//【GECC】：椭圆曲线密码术指南（汉克森、门尼泽、万通）

import (
	"encoding/binary"
	"math/big"
)

//
//
var secp256k1BytePoints = ""

//GetDoublingPoints返回所有可能的g^（2^i）for i in
//0..n-1，其中n是曲线的位大小（对于secp256k1为256）
//坐标记录为雅可比坐标。
func (curve *KoblitzCurve) getDoublingPoints() [][3]fieldVal {
	doublingPoints := make([][3]fieldVal, curve.BitSize)

//
	px, py := curve.bigAffineToField(curve.Gx, curve.Gy)
	pz := new(fieldVal).SetInt(1)
	for i := 0; i < curve.BitSize; i++ {
		doublingPoints[i] = [3]fieldVal{*px, *py, *pz}
//
		curve.doubleJacobian(px, py, pz, px, py, pz)
	}
	return doublingPoints
}

//
//
//
func (curve *KoblitzCurve) SerializedBytePoints() []byte {
	doublingPoints := curve.getDoublingPoints()

//
	serialized := make([]byte, curve.byteSize*256*3*10*4)
	offset := 0
	for byteNum := 0; byteNum < curve.byteSize; byteNum++ {
//
		startingBit := 8 * (curve.byteSize - byteNum - 1)
		computingPoints := doublingPoints[startingBit : startingBit+8]

//
		for i := 0; i < 256; i++ {
			px, py, pz := new(fieldVal), new(fieldVal), new(fieldVal)
			for j := 0; j < 8; j++ {
				if i>>uint(j)&1 == 1 {
					curve.addJacobian(px, py, pz, &computingPoints[j][0],
						&computingPoints[j][1], &computingPoints[j][2], px, py, pz)
				}
			}
			for i := 0; i < 10; i++ {
				binary.LittleEndian.PutUint32(serialized[offset:], px.n[i])
				offset += 4
			}
			for i := 0; i < 10; i++ {
				binary.LittleEndian.PutUint32(serialized[offset:], py.n[i])
				offset += 4
			}
			for i := 0; i < 10; i++ {
				binary.LittleEndian.PutUint32(serialized[offset:], pz.n[i])
				offset += 4
			}
		}
	}

	return serialized
}

//
//
//
func sqrt(n *big.Int) *big.Int {
//初始猜测=2^（对数2（n）/2）
	guess := big.NewInt(2)
	guess.Exp(guess, big.NewInt(int64(n.BitLen()/2)), nil)

//现在使用牛顿的方法进行改进。
	big2 := big.NewInt(2)
	prevGuess := big.NewInt(0)
	for {
		prevGuess.Set(guess)
		guess.Add(guess, new(big.Int).Div(n, guess))
		guess.Div(guess, big2)
		if guess.Cmp(prevGuess) == 0 {
			break
		}
	}
	return guess
}

//自同态向量运行算法3.74的前3个步骤，从[GECC]到
//生成生成平衡所需的线性无关向量
//
//
//
//
func (curve *KoblitzCurve) EndomorphismVectors() (a1, b1, a2, b2 *big.Int) {
	bigMinus1 := big.NewInt(-1)

//
//
//

	nSqrt := sqrt(curve.N)
	u, v := new(big.Int).Set(curve.N), new(big.Int).Set(curve.lambda)
	x1, y1 := big.NewInt(1), big.NewInt(0)
	x2, y2 := big.NewInt(0), big.NewInt(1)
	q, r := new(big.Int), new(big.Int)
	qu, qx1, qy1 := new(big.Int), new(big.Int), new(big.Int)
	s, t := new(big.Int), new(big.Int)
	ri, ti := new(big.Int), new(big.Int)
	a1, b1, a2, b2 = new(big.Int), new(big.Int), new(big.Int), new(big.Int)
	found, oneMore := false, false
	for u.Sign() != 0 {
//
		q.Div(v, u)

//
		qu.Mul(q, u)
		r.Sub(v, qu)

//
		qx1.Mul(q, x1)
		s.Sub(x2, qx1)

//t= y2-q*y1
		qy1.Mul(q, y1)
		t.Sub(y2, qy1)

//v=u，u=r，x2=x1，x1=s，y2=y1，y1=t
		v.Set(u)
		u.Set(r)
		x2.Set(x1)
		x1.Set(s)
		y2.Set(y1)
		y1.Set(t)

//当余数小于n的sqrt时，
//
		if !found && r.Cmp(nSqrt) < 0 {
//
//
//
//

//
			a1.Set(r)
			b1.Mul(t, bigMinus1)
			found = true
			oneMore = true

//
//被改进的。
			continue

		} else if oneMore {
//
//表示r[i]和t[i]值，而当前
//r和t分别为r[i+2]和t[i+2]。

//sum1 = r[i]^2 + t[i]^2
			rSquared := new(big.Int).Mul(ri, ri)
			tSquared := new(big.Int).Mul(ti, ti)
			sum1 := new(big.Int).Add(rSquared, tSquared)

//sum2=r[i+2]^2+t[i+2]^2
			r2Squared := new(big.Int).Mul(r, r)
			t2Squared := new(big.Int).Mul(t, t)
			sum2 := new(big.Int).Add(r2Squared, t2Squared)

//如果（r[i]^2+t[i]^2）<=（r[i+2]^2+t[i+2]^2）
			if sum1.Cmp(sum2) <= 0 {
//a2=r[i]，b2=-t[i]
				a2.Set(ri)
				b2.Mul(ti, bigMinus1)
			} else {
//a2=r[i+2]，b2=-t[i+2]
				a2.Set(r)
				b2.Mul(t, bigMinus1)
			}

//都做完了。
			break
		}

		ri.Set(r)
		ti.Set(t)
	}

	return a1, b1, a2, b2
}
