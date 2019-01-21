
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有2010 Go作者。版权所有。
//版权所有2011 Thepiachu。版权所有。
//版权所有2013-2014 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package btcec

//参考文献：
//【secg】：建议椭圆曲线域参数
//http://www.secg.org/sec2-v2.pdf
//
//【GECC】：椭圆曲线密码术指南（汉克森、门尼泽、万通）

//这个包在内部以雅可比坐标运行。对于给定的
//（x，y）曲线上的位置，雅可比坐标为（x1，y1，z1）
//其中x=x1/z1？和y=y1/z1？3。当整个过程
//可以在转换中执行计算（如scalarmult和
//scalarbasemult）。但即使是加法器和双倍加法器，应用和
//将转换反转为仿射坐标。

import (
	"crypto/elliptic"
	"math/big"
	"sync"
)

var (
//FieldOne只是字段表示中的整数1。它是
//用于避免在内部
//算术。
	fieldOne = new(fieldVal).SetInt(1)
)

//koblitz curve支持符合ECC曲线的koblitz曲线实现
//来自crypto/椭圆的接口。
type KoblitzCurve struct {
	*elliptic.CurveParams
	q         *big.Int
H         int      //曲线的辅因子。
halfOrder *big.Int //一半订单N

//bytesize只是位大小/8，为方便起见而提供
//因为它是重复计算的。
	byteSize int

//旁注点
	bytePoints *[32][256][3]fieldVal

//接下来的6个值专门用于自同态
//scalarmult中的优化。

//lambda必须满足lambda^3=1 mod n，其中n是g的顺序。
	lambda *big.Int

//beta必须满足beta^3=1 mod p，其中p是
//曲线。
	beta *fieldVal

//参见gensecp256k1中的自同态向量。去看看它们是如何的。
//衍生的。
	a1 *big.Int
	b1 *big.Int
	a2 *big.Int
	b2 *big.Int
}

//params返回曲线的参数。
func (curve *KoblitzCurve) Params() *elliptic.CurveParams {
	return curve.CurveParams
}

//bigaffinetofield将仿射点（x，y）作为大整数并转换
//它作为字段值到达仿射点。
func (curve *KoblitzCurve) bigAffineToField(x, y *big.Int) (*fieldVal, *fieldVal) {
	x3, y3 := new(fieldVal), new(fieldVal)
	x3.SetByteSlice(x.Bytes())
	y3.SetByteSlice(y.Bytes())

	return x3, y3
}

//fieldjacobiantobigaffine以jacobian点（x，y，z）作为字段值和
//将其转换为仿射点作为大整数。
func (curve *KoblitzCurve) fieldJacobianToBigAffine(x, y, z *fieldVal) (*big.Int, *big.Int) {
//倒转成本很高，而且增加了点和增加了点
//使用Z值为1的点时速度更快。所以，
//如果需要将点转换为仿射，则继续进行并规范化
//点本身与计算的同时也是一样的。
	var zInv, tempZ fieldVal
zInv.Set(z).Inverse()   //Ziv= Z^—1
tempZ.SquareVal(&zInv)  //TZZZ＝Z^—2
x.Mul(&tempZ)           //x=x/z^2（磁：1）
y.Mul(tempZ.Mul(&zInv)) //Y=Y/Z^3（磁：1）
z.SetInt(1)             //Z＝1（MAG：1）

//规格化x和y值。
	x.Normalize()
	y.Normalize()

//将现在仿射点的字段值转换为big.ints。
	x3, y3 := new(big.Int), new(big.Int)
	x3.SetBytes(x.Bytes()[:])
	y3.SetBytes(y.Bytes()[:])
	return x3, y3
}

//如果点（x，y）在曲线上，is on curve返回布尔值。
//椭圆曲线接口的一部分。此函数与
//加密/椭圆算法，因为a=0而不是-3。
func (curve *KoblitzCurve) IsOnCurve(x, y *big.Int) bool {
//将大整数转换为字段值以实现更快的算术运算。
	fx, fy := curve.bigAffineToField(x, y)

//secp256k1的椭圆曲线方程为：y^2=x^3+7
	y2 := new(fieldVal).SquareVal(fy).Normalize()
	result := new(fieldVal).SquareVal(fx).Mul(fx).AddInt(7).Normalize()
	return y2.Equals(result)
}

//addz1和z2equalsone增加了两个已知的雅可比点
//z值为1，结果存储在（x3、y3、z3）中。这就是说
//（x1，y1，1）+（x2，y2，1）=（x3，y3，z3）。它比
//一般的添加例程，因为由于能够
//避免Z值乘法。
func (curve *KoblitzCurve) addZ1AndZ2EqualsOne(x1, y1, z1, x2, y2, x3, y3, z3 *fieldVal) {
//为了有效地计算点加法，该实现将
//将方程转化为用于最小化的中间元素
//使用如下所示方法的字段乘法数：
//http://hyper椭圆形.org/efd/g1p/auto-shortw-jacobian-0.html addition-mmad-2007-bl
//
//
//h=x2-x1，h h=h^2，i=4*h h，j=h*i，r=2*（y2-y1），v=x1*i
//x3=r^2-j-2*v，y3=r*（v-x3）-2*y1*j，z3=2*h
//
//这会导致4次场乘，2次场平方，
//6个字段加法和5个整数乘法。

//当曲线上两个点的X坐标相同时，
//Y坐标必须相同，在这种情况下，它是点。
//加倍，或者相反，结果是
//椭圆曲线密码体制的无穷大。
	x1.Normalize()
	y1.Normalize()
	x2.Normalize()
	y2.Normalize()
	if x1.Equals(x2) {
		if y1.Equals(y2) {
//由于x1==x2和y1==y2，点加倍必须
//完成，否则加法将以除法结束
//零度。
			curve.doubleJacobian(x1, y1, z1, x3, y3, z3)
			return
		}

//因为x1==x2和y1=-y2，所以和是
//根据群律无穷大。
		x3.SetInt(0)
		y3.SetInt(0)
		z3.SetInt(0)
		return
	}

//根据中间元素计算x3、y3和z3
//分解以上。
	var h, i, j, r, v fieldVal
	var negJ, neg2V, negX3 fieldVal
h.Set(x1).Negate(1).Add(x2)                //H=x2-x1（mag:3）
i.SquareVal(&h).MulInt(4)                  //i=4*h^2（磁：4）
j.Mul2(&h, &i)                             //J=H*I（磁：1）
r.Set(y1).Negate(1).Add(y2).MulInt(2)      //R=2*（Y2-Y1）（磁：6）
v.Mul2(x1, &i)                             //V=x1*i（磁：1）
negJ.Set(&j).Negate(1)                     //negj=-j（mag:2）
neg2V.Set(&v).MulInt(2).Negate(2)          //neg2v=-（2*v）（mag:3）
x3.Set(&r).Square().Add(&negJ).Add(&neg2V) //
negX3.Set(x3).Negate(6)                    //negx3=-x3（磁：7）
j.Mul(y1).MulInt(2).Negate(2)              //J=—（2*y1*j）（mag:3）
y3.Set(&v).Add(&negX3).Mul(&r).Add(&j)     //y3=r*（v-x3）-2*y1*j（mag:4）
z3.Set(&h).MulInt(2)                       //Z3=2*h（磁：6）

//根据需要将生成的字段值规范化为1的大小。
	x3.Normalize()
	y3.Normalize()
	z3.Normalize()
}

//addz1equalsz2添加了两个已知具有
//相同的z值并将结果存储在（x3、y3、z3）中。这就是说
//（x1，y1，z1）+（x2，y2，z1）=（x3，y3，z3）。它比
//
//等价性。
func (curve *KoblitzCurve) addZ1EqualsZ2(x1, y1, z1, x2, y2, x3, y3, z3 *fieldVal) {
//为了有效地计算点加法，该实现将
//将方程转化为用于最小化的中间元素
//使用稍微修改过的版本的字段乘法数
//方法如下：
//http://hyper椭圆形.org/efd/g1p/auto-shortw-jacobian-0.html addition-mmad-2007-bl
//
//特别是，它使用以下方法执行计算：
//a=x2-x1，b=a^2，c=y2-y1，d=c^2，e=x1*b，f=x2*b
//x3=d-e-f，y3=c*（e-x3）-y1*（f-e），z3=z1*a
//
//这会导致5次场乘，2次场平方，
//9个字段相加，0个整数乘法。

//当曲线上两个点的X坐标相同时，
//Y坐标必须相同，在这种情况下，它是点。
//加倍，或者相反，结果是
//椭圆曲线密码体制的无穷大。
	x1.Normalize()
	y1.Normalize()
	x2.Normalize()
	y2.Normalize()
	if x1.Equals(x2) {
		if y1.Equals(y2) {
//由于x1==x2和y1==y2，点加倍必须
//完成，否则加法将以除法结束
//零度。
			curve.doubleJacobian(x1, y1, z1, x3, y3, z3)
			return
		}

//因为x1==x2和y1=-y2，所以和是
//根据群律无穷大。
		x3.SetInt(0)
		y3.SetInt(0)
		z3.SetInt(0)
		return
	}

//根据中间元素计算x3、y3和z3
//分解以上。
	var a, b, c, d, e, f fieldVal
	var negX1, negY1, negE, negX3 fieldVal
negX1.Set(x1).Negate(1)                //negx1=-x1（mag:2）
negY1.Set(y1).Negate(1)                //negy1=-y1（mag:2）
a.Set(&negX1).Add(x2)                  //A=x2-x1（mag:3）
b.SquareVal(&a)                        //
c.Set(&negY1).Add(y2)                  //C=Y2-Y1（磁：3）
d.SquareVal(&c)                        //D=C^2（磁：1）
e.Mul2(x1, &b)                         //E=x1*b（磁：1）
negE.Set(&e).Negate(1)                 //nege=-e（mag:2）
f.Mul2(x2, &b)                         //F=x2*b（磁：1）
x3.Add2(&e, &f).Negate(3).Add(&d)      //x3=d-e-f（mag:5）
negX3.Set(x3).Negate(5).Normalize()    //negx3=-x3（磁：1）
y3.Set(y1).Mul(f.Add(&negE)).Negate(3) //Y3=—（y1*（f-e））（mag:4）
y3.Add(e.Add(&negX3).Mul(&c))          //y3=c*（e-x3）+y3（mag:5）
z3.Mul2(z1, &a)                        //Z3=Z1*A（磁：1）

//根据需要将生成的字段值规范化为1的大小。
	x3.Normalize()
	y3.Normalize()
}

//当第二个点已经存在时，addz2equalsone加上两个雅可比点。
//已知Z值为1（第一个点的Z值不是1）
//并将结果存储在（x3、y3、z3）中。也就是说（x1，y1，z1）+
//（x2，y2，1）=（x3，y3，z3）。它执行的加法比一般的快
//添加例程，因为由于能够避免
//乘以第二个点的z值。
func (curve *KoblitzCurve) addZ2EqualsOne(x1, y1, z1, x2, y2, x3, y3, z3 *fieldVal) {
//为了有效地计算点加法，该实现将
//将方程转化为用于最小化的中间元素
//使用如下所示方法的字段乘法数：
//http://hyper椭圆形.org/efd/g1p/auto-shortw-jacobian-0.html addition-madd-2007-bl
//
//特别是，它使用以下方法执行计算：
//z1 z1=z1^2，u2=x2*z1 z1，s2=y2*z1*z1 z1，h=u2-x1，h h=h^2，
//i=4*h h，j=h*i，r=2*（s2-y1），v=x1*i
//x3=r^2-j-2*v，y3=r*（v-x3）-2*y1*j，z3=（z1+h）^2-z1 z1-h h
//
//这将导致7次场乘，4次场平方，
//9个字段加法和4个整数乘法。

//当曲线上两个点的X坐标相同时，
//Y坐标必须相同，在这种情况下，它是点。
//加倍，或者相反，结果是
//椭圆曲线密码体制的无穷大。自从
//任何数量的雅可比坐标都可以表示相同的仿射。
//点，x和y值需要转换成类似的条件。由于
//对这个函数所作的假设是，第二个点有一个z
//值1（z2=1），第一个点已经“转换”。
	var z1z1, u2, s2 fieldVal
	x1.Normalize()
	y1.Normalize()
z1z1.SquareVal(z1)                        //z1 z1=z1^2（磁：1）
u2.Set(x2).Mul(&z1z1).Normalize()         //u2=x2*z1z1（磁：1）
s2.Set(y2).Mul(&z1z1).Mul(z1).Normalize() //s2=y2*z1*z1 z1（mag:1）
	if x1.Equals(&u2) {
		if y1.Equals(&s2) {
//由于x1==x2和y1==y2，点加倍必须
//完成，否则加法将以除法结束
//零度。
			curve.doubleJacobian(x1, y1, z1, x3, y3, z3)
			return
		}

//因为x1==x2和y1=-y2，所以和是
//根据群律无穷大。
		x3.SetInt(0)
		y3.SetInt(0)
		z3.SetInt(0)
		return
	}

//根据中间元素计算x3、y3和z3
//分解以上。
	var h, hh, i, j, r, rr, v fieldVal
	var negX1, negY1, negX3 fieldVal
negX1.Set(x1).Negate(1)                //negx1=-x1（mag:2）
h.Add2(&u2, &negX1)                    //H=u2-x1（磁：3）
hh.SquareVal(&h)                       //h h=h^2（mag:1）
i.Set(&hh).MulInt(4)                   //I=4*hh（磁：4）
j.Mul2(&h, &i)                         //J=H*I（磁：1）
negY1.Set(y1).Negate(1)                //negy1=-y1（mag:2）
r.Set(&s2).Add(&negY1).MulInt(2)       //r=2*（s2-y1）（磁：6）
rr.SquareVal(&r)                       //RR=R^2（磁：1）
v.Mul2(x1, &i)                         //V=x1*i（磁：1）
x3.Set(&v).MulInt(2).Add(&j).Negate(3) //x3=—（J+2*V）（磁：4）
x3.Add(&rr)                            //x3=r^2+x3（mag:5）
negX3.Set(x3).Negate(5)                //negx3=-x3（磁：6）
y3.Set(y1).Mul(&j).MulInt(2).Negate(2) //Y3=—（2*y1*j）（mag:3）
y3.Add(v.Add(&negX3).Mul(&r))          //Y3=R*（V-x3）+Y3（磁：4）
z3.Add2(z1, &h).Square()               //z3=（z1+h）^2（mag:1）
z3.Add(z1z1.Add(&hh).Negate(2))        //z3=z3-（z1z1+hh）（mag:4）

//根据需要将生成的字段值规范化为1的大小。
	x3.Normalize()
	y3.Normalize()
	z3.Normalize()
}

//addgeneric添加了两个雅可比点（x1，y1，z1）和（x2，y2，z2），而不添加任何雅可比点
//假设两点的z值并将结果存储在
//（X3，Y3，Z3）。也就是说（x1，y1，z1）+（x2，y2，z2）=（x3，y3，z3）。它
//是添加例程中最慢的，因为需要的算术量最大。
func (curve *KoblitzCurve) addGeneric(x1, y1, z1, x2, y2, z2, x3, y3, z3 *fieldVal) {
//为了有效地计算点加法，该实现将
//将方程转化为用于最小化的中间元素
//使用如下所示方法的字段乘法数：
//http://hyper椭圆形.org/efd/g1p/auto-shortw-jacobian-0.html addition-add-2007-bl
//
//特别是，它使用以下方法执行计算：
//z1 z1=z1^2，z2 z2=z2^2，u1=x1*z2 z2，u2=x2*z1 z1，s1=y1*z2*z2 z2
//s2=y2*z1*z1 z1，h=u2-u1，i=（2*h）^2，j=h*i，r=2*（s2-s1）
//V= U1*I
//x3=r^2-j-2*v，y3=r*（v-x3）-2*s1*j，z3=（z1+z2）^2-z1 z1-z2 z2）*h
//
//这会导致11次场乘，5次场平方，
//9个字段加法和4个整数乘法。

//当曲线上两个点的X坐标相同时，
//Y坐标必须相同，在这种情况下，它是点。
//加倍，或者相反，结果是
//无穷。因为任何数量的雅可比坐标都可以表示
//相同的仿射点，x和y值需要转换为like
//条款。
	var z1z1, z2z2, u1, u2, s1, s2 fieldVal
z1z1.SquareVal(z1)                        //z1 z1=z1^2（磁：1）
z2z2.SquareVal(z2)                        //z2 z2=z2^2（mag:1）
u1.Set(x1).Mul(&z2z2).Normalize()         //u1=x1*z2z2（mag:1）
u2.Set(x2).Mul(&z1z1).Normalize()         //u2=x2*z1z1（磁：1）
s1.Set(y1).Mul(&z2z2).Mul(z2).Normalize() //s1=y1*z2*z2 z2（mag:1）
s2.Set(y2).Mul(&z1z1).Mul(z1).Normalize() //s2=y2*z1*z1 z1（mag:1）
	if u1.Equals(&u2) {
		if s1.Equals(&s2) {
//由于x1==x2和y1==y2，点加倍必须
//完成，否则加法将以除法结束
//零度。
			curve.doubleJacobian(x1, y1, z1, x3, y3, z3)
			return
		}

//因为x1==x2和y1=-y2，所以和是
//根据群律无穷大。
		x3.SetInt(0)
		y3.SetInt(0)
		z3.SetInt(0)
		return
	}

//根据中间元素计算x3、y3和z3
//分解以上。
	var h, i, j, r, rr, v fieldVal
	var negU1, negS1, negX3 fieldVal
negU1.Set(&u1).Negate(1)               //negu1=-u1（mag:2）
h.Add2(&u2, &negU1)                    //H=U2-U1（磁：3）
i.Set(&h).MulInt(2).Square()           //i=（2*h）^2（mag:2）
j.Mul2(&h, &i)                         //J=H*I（磁：1）
negS1.Set(&s1).Negate(1)               //negs1=-s1（mag:2）
r.Set(&s2).Add(&negS1).MulInt(2)       //r=2*（s2-s1）（mag:6）
rr.SquareVal(&r)                       //RR=R^2（磁：1）
v.Mul2(&u1, &i)                        //V=U1*I（磁：1）
x3.Set(&v).MulInt(2).Add(&j).Negate(3) //x3=—（J+2*V）（磁：4）
x3.Add(&rr)                            //x3=r^2+x3（mag:5）
negX3.Set(x3).Negate(5)                //negx3=-x3（磁：6）
y3.Mul2(&s1, &j).MulInt(2).Negate(2)   //Y3=—（2*s1*j）（mag:3）
y3.Add(v.Add(&negX3).Mul(&r))          //Y3=R*（V-x3）+Y3（磁：4）
z3.Add2(z1, z2).Square()               //z3=（z1+z2）^2（mag:1）
z3.Add(z1z1.Add(&z2z2).Negate(2))      //z3=z3-（z1z1+z2z2）（mag:4）
z3.Mul(&h)                             //Z3=Z3*H（磁：1）

//根据需要将生成的字段值规范化为1的大小。
	x3.Normalize()
	y3.Normalize()
}

//AddJacobian将传递的Jacobian点（x1、y1、z1）和（x2、y2、z2）相加。
//并将结果存储在（x3、y3、z3）中。
func (curve *KoblitzCurve) addJacobian(x1, y1, z1, x2, y2, z2, x3, y3, z3 *fieldVal) {
//
//椭圆曲线密码术。因此，∞+p=p和p+∞=p。
	if (x1.IsZero() && y1.IsZero()) || z1.IsZero() {
		x3.Set(x2)
		y3.Set(y2)
		z3.Set(z2)
		return
	}
	if (x2.IsZero() && y2.IsZero()) || z2.IsZero() {
		x3.Set(x1)
		y3.Set(y1)
		z3.Set(z1)
		return
	}

//当某些假设为
//遇见。例如，当两个点具有相同的Z值时，算术
//可以避免Z值。因此，本节检查这些
//条件并调用适当的加法函数
//使用这些假设。
	z1.Normalize()
	z2.Normalize()
	isZ1One := z1.Equals(fieldOne)
	isZ2One := z2.Equals(fieldOne)
	switch {
	case isZ1One && isZ2One:
		curve.addZ1AndZ2EqualsOne(x1, y1, z1, x2, y2, x3, y3, z3)
		return
	case z1.Equals(z2):
		curve.addZ1EqualsZ2(x1, y1, z1, x2, y2, x3, y3, z3)
		return
	case isZ2One:
		curve.addZ2EqualsOne(x1, y1, z1, x2, y2, x3, y3, z3)
		return
	}

//以上假设都不是真的，所以回到一般的假设。
//点加法
	curve.addGeneric(x1, y1, z1, x2, y2, z2, x3, y3, z3)
}

//
//接口。
func (curve *KoblitzCurve) Add(x1, y1, x2, y2 *big.Int) (*big.Int, *big.Int) {
//
//椭圆曲线密码术。因此，∞+p=p和p+∞=p。
	if x1.Sign() == 0 && y1.Sign() == 0 {
		return x2, y2
	}
	if x2.Sign() == 0 && y2.Sign() == 0 {
		return x1, y1
	}

//将仿射坐标从大整数转换为字段值
//在雅可比射影空间中做点加法。
	fx1, fy1 := curve.bigAffineToField(x1, y1)
	fx2, fy2 := curve.bigAffineToField(x2, y2)
	fx3, fy3, fz3 := new(fieldVal), new(fieldVal), new(fieldVal)
	fOne := new(fieldVal).SetInt(1)
	curve.addJacobian(fx1, fy1, fOne, fx2, fy2, fOne, fx3, fy3, fz3)

//将雅可比坐标场值转换回仿射大
//整数。
	return curve.fieldJacobianToBigAffine(fx3, fy3, fz3)
}

//
//当已知点的z值为1并存储时
//结果是（x3，y3，z3）。也就是说（x3，y3，z3）=2*（x1，y1，1）。它
//与常规例程相比，执行更快的点加倍，因为算法更少
//
func (curve *KoblitzCurve) doubleZ1EqualsOne(x1, y1, x3, y3, z3 *fieldVal) {
//此函数使用假设z1为1，因此点
//加倍公式减少到：
//
//x3=（3*x1^2）^2-8*x1*y1^2
//Y3=（3*x1^2）*（4*x1*y1^2-x3）-8*y1^4
//Z3＝2＊Y1
//
//为了有效地计算上述内容，此实现将
//
//有利于场平方的场倍增数
//比场与电流的相乘快大约35%
//编写时的实现。
//
//这将使用稍微修改过的方法版本，如下所示：
//http://hyper椭圆形.org/efd/g1p/auto-shortw-jacobian-0.html doubling-mdbl-2007-bl
//
//特别是，它使用以下方法执行计算：
//a=x1^2，b=y1^2，c=b^2，d=2*（（x1+b）^2-a-c）
//e=3*a，f=e^2，x3=f-2*d，y3=e*（d-x3）-8*c
//Z3＝2＊Y1
//
//这将导致1场乘法、5场平方运算的开销，
//6个字段加法和5个整数乘法。
	var a, b, c, d, e, f fieldVal
z3.Set(y1).MulInt(2)                     //Z3=2*Y1（磁：2）
a.SquareVal(x1)                          //A=x1^2（磁：1）
b.SquareVal(y1)                          //B=y1^2（磁：1）
c.SquareVal(&b)                          //C=B^2（磁：1）
b.Add(x1).Square()                       //b=（x1+b）^2（mag:1）
d.Set(&a).Add(&c).Negate(2)              //D=—（A+C）（磁：3）
d.Add(&b).MulInt(2)                      //d=2*（b+d）（mag:8）
e.Set(&a).MulInt(3)                      //E=3*A（磁：3）
f.SquareVal(&e)                          //
x3.Set(&d).MulInt(2).Negate(16)          //x3=—（2*d）（mag:17）
x3.Add(&f)                               //x3=f+x3（mag:18）
f.Set(x3).Negate(18).Add(&d).Normalize() //F=D-x3（磁：1）
y3.Set(&c).MulInt(8).Negate(8)           //Y3=—（8*C）（mag:9）
y3.Add(f.Mul(&e))                        //Y3=E*F+Y3（磁：10）

//将字段值规格化回1的大小。
	x3.Normalize()
	y3.Normalize()
	z3.Normalize()
}

//doublegeneric对传递的Jacobian点执行点加倍，而不
//关于z值的任何假设，并将结果存储在（x3、y3、z3）中。
//也就是说（x3，y3，z3）=2*（x1，y1，z1）。这是最慢的一点
//
func (curve *KoblitzCurve) doubleGeneric(x1, y1, z1, x3, y3, z3 *fieldVal) {
//secp256k1雅可比坐标的点加倍公式
//曲线：
//x3=（3*x1^2）^2-8*x1*y1^2
//Y3=（3*x1^2）*（4*x1*y1^2-x3）-8*y1^4
//Z3＝2＊Y1*Z1
//
//为了有效地计算上述内容，此实现将
//方程式转化为中间元素，用于最小化
//有利于场平方的场倍增数
//比场与电流的相乘快大约35%
//编写时的实现。
//
//这将使用稍微修改过的方法版本，如下所示：
//http://hyper椭圆形.org/efd/g1p/auto-shortw-jacobian-0.html doubling-dbl-2009-l
//
//特别是，它使用以下方法执行计算：
//a=x1^2，b=y1^2，c=b^2，d=2*（（x1+b）^2-a-c）
//e=3*a，f=e^2，x3=f-2*d，y3=e*（d-x3）-8*c
//Z3＝2＊Y1*Z1
//
//这将导致1场乘法、5场平方运算的开销，
//6个字段加法和5个整数乘法。
	var a, b, c, d, e, f fieldVal
z3.Mul2(y1, z1).MulInt(2)                //z3=2*y1*z1（mag:2）
a.SquareVal(x1)                          //A=x1^2（磁：1）
b.SquareVal(y1)                          //B=y1^2（磁：1）
c.SquareVal(&b)                          //C=B^2（磁：1）
b.Add(x1).Square()                       //b=（x1+b）^2（mag:1）
d.Set(&a).Add(&c).Negate(2)              //D=—（A+C）（磁：3）
d.Add(&b).MulInt(2)                      //d=2*（b+d）（mag:8）
e.Set(&a).MulInt(3)                      //E=3*A（磁：3）
f.SquareVal(&e)                          //F=E^2（磁：1）
x3.Set(&d).MulInt(2).Negate(16)          //x3=—（2*d）（mag:17）
x3.Add(&f)                               //x3=f+x3（mag:18）
f.Set(x3).Negate(18).Add(&d).Normalize() //F=D-x3（磁：1）
y3.Set(&c).MulInt(8).Negate(8)           //Y3=—（8*C）（mag:9）
y3.Add(f.Mul(&e))                        //Y3=E*F+Y3（磁：10）

//将字段值规格化回1的大小。
	x3.Normalize()
	y3.Normalize()
	z3.Normalize()
}

//doublejacobian将传递的Jacobian点（x1、y1、z1）加倍，并存储
//
func (curve *KoblitzCurve) doubleJacobian(x1, y1, z1, x3, y3, z3 *fieldVal) {
//
	if y1.IsZero() || z1.IsZero() {
		x3.SetInt(0)
		y3.SetInt(0)
		z3.SetInt(0)
		return
	}

//当z值为1时，可以更快地实现点倍增。
//
//
//
	if z1.Normalize().Equals(fieldOne) {
		curve.doubleZ1EqualsOne(x1, y1, x3, y3, z3)
		return
	}

//返回到使用任意z的通用点加倍
//价值观。
	curve.doubleGeneric(x1, y1, z1, x3, y3, z3)
}

//double返回2*（x1，y1）。椭圆曲线接口的一部分。
func (curve *KoblitzCurve) Double(x1, y1 *big.Int) (*big.Int, *big.Int) {
	if y1.Sign() == 0 {
		return new(big.Int), new(big.Int)
	}

//将仿射坐标从大整数转换为字段值
//
	fx1, fy1 := curve.bigAffineToField(x1, y1)
	fx3, fy3, fz3 := new(fieldVal), new(fieldVal), new(fieldVal)
	fOne := new(fieldVal).SetInt(1)
	curve.doubleJacobian(fx1, fy1, fOne, fx3, fy3, fz3)

//将雅可比坐标场值转换回仿射大
//整数。
	return curve.fieldJacobianToBigAffine(fx3, fy3, fz3)
}

//splitk返回一个平衡长度的k及其符号的两种表示形式。
//这是来自[GECC]的算法3.74。
//
//关于这个算法，需要注意的一点是，无论c1和c2是什么，
//k=k1+k2*lambda（mod n）的最终方程式成立。这是
//从数学上证明A1/B1/A2/B2是如何计算的。
//
//选择c1和c2以最小化最大值（k1、k2）。
func (curve *KoblitzCurve) splitK(k []byte) ([]byte, []byte, int, int) {
//这里所有的数学都是用big.int完成的，它很慢。
//在某种程度上，写一些类似于
//fieldval，但如果结束，则将n而不是p作为主字段
//
	bigIntK := new(big.Int)
	c1, c2 := new(big.Int), new(big.Int)
	tmp1, tmp2 := new(big.Int), new(big.Int)
	k1, k2 := new(big.Int), new(big.Int)

	bigIntK.SetBytes(k)
//c1=步骤4中的圆形（b2*k/n）。
//四舍五入并非真正必要，而且成本太高，因此跳过了
	c1.Mul(curve.b2, bigIntK)
	c1.Div(c1, curve.N)
//c2=从步骤4开始的圆形（b1*k/n）（符号颠倒以优化一个步骤）
//四舍五入并非真正必要，而且成本太高，因此跳过了
	c2.Mul(curve.b1, bigIntK)
	c2.Div(c2, curve.N)
//步骤5的k1=k-c1*a1-c2*a2（注c2的符号相反）
	tmp1.Mul(c1, curve.a1)
	tmp2.Mul(c2, curve.a2)
	k1.Sub(bigIntK, tmp1)
	k1.Add(k1, tmp2)
//
	tmp1.Mul(c1, curve.b1)
	tmp2.Mul(c2, curve.b2)
	k2.Sub(tmp2, tmp1)

//注意bytes（）抛出k1和k2的符号。这件事
//因为k1和/或k2可以是负的。因此，我们通过了
//分开回来。
	return k1.Bytes(), k2.Bytes(), k1.Sign(), k2.Sign()
}

//moduledUCE将k从32字节以上减少到32字节以下。这个
//
//因此椭圆曲线上的其他有效点具有相同的顺序。
func (curve *KoblitzCurve) moduloReduce(k []byte) []byte {
//因为g的阶是曲线n，所以我们可以用一个小得多的数。
//
	if len(k) > curve.byteSize {
//通过执行模曲线减少k。
		tmpK := new(big.Int).SetBytes(k)
		tmpK.Mod(tmpK, curve.N)
		return tmpK.Bytes()
	}

	return k
}

//naf取一个正整数k，并将非相邻形式（naf）返回为2
//字节切片。第一个是1s的位置。第二个是-1的位置
//是。NAF比较方便，平均只有1/3的值是
//非零。这是来自[GECC]的算法3.30。
//
//从本质上讲，这使得最小化操作的数量成为可能。
//因为返回的结果int至少为50%0s。
func NAF(k []byte) ([]byte, []byte) {
//这个算法的本质是，每当我们有连续的1
//在二进制文件中，我们想把-1放在最低位，得到一组
//0到连续1的最高位。这是由于
//身份：
//2^n+2^（n-1）+2^（n-2）+…+2^（n-k）=2^（n+1）-2^（n-k）
//
//因此，该算法可能需要比
//我们实际上有比特，因此比特比以前长了1比特。
//必要的。因为我们需要知道加法是否会导致进位，
//我们再从右向左走。
	var carry, curIsOne, nextIsOne bool
//这些默认值为零
	retPos := make([]byte, len(k)+1)
	retNeg := make([]byte, len(k)+1)
	for i := len(k) - 1; i >= 0; i-- {
		curByte := k[i]
		for j := uint(0); j < 8; j++ {
			curIsOne = curByte&1 == 1
			if j == 7 {
				if i == 0 {
					nextIsOne = false
				} else {
					nextIsOne = k[i-1]&1 == 1
				}
			} else {
				nextIsOne = curByte&2 == 2
			}
			if carry {
				if curIsOne {
//这个位是1，所以继续进位
//不需要做任何事。
				} else {
//我们在几次
//1s。
					if nextIsOne {
//从那以后再带一次
//新的1s序列是
//启动。
						retNeg[i+1] += 1 << j
					} else {
//从1开始停止携带
//停止。
						carry = false
						retPos[i+1] += 1 << j
					}
				}
			} else if curIsOne {
				if nextIsOne {
//如果这是至少2个的开始
//连续1秒，设置当前1秒
//到-1开始携带。
					retNeg[i+1] += 1 << j
					carry = true
				} else {
//这是单打，不是连续的
//1s。
					retPos[i+1] += 1 << j
				}
			}
			curByte >>= 1
		}
	}
	if carry {
		retPos[0] = 1
		return retPos, retNeg
	}
	return retPos[1:], retNeg[1:]
}

//
//椭圆曲线接口的一部分。
func (curve *KoblitzCurve) ScalarMult(Bx, By *big.Int, k []byte) (*big.Int, *big.Int) {
//点Q=∞（无穷远点）。
	qx, qy, qz := new(fieldVal), new(fieldVal), new(fieldVal)

//将K分解成k1和k2，使EC操作数减半。
//参见[GECC]中的算法3.74。
	k1, k2, signK1, signK2 := curve.splitK(curve.moduloReduce(k))

//这里要记住的主要方程式是：
//
//
//式中p1以下为p，式中p2以下为_（p）
	p1x, p1y := curve.bigAffineToField(Bx, By)
	p1yNeg := new(fieldVal).NegateVal(p1y, 1)
	p1z := new(fieldVal).SetInt(1)

//注：⑨（x，y）=（βx，y）。雅可比z坐标是1，所以这个数学
//通过。
	p2x := new(fieldVal).Mul2(p1x, curve.beta)
	p2y := new(fieldVal).Set(p1y)
	p2yNeg := new(fieldVal).NegateVal(p2y, 1)
	p2z := new(fieldVal).SetInt(1)

//根据需要翻转点的正值和负值
//取决于k1和k2的符号。如等式所述
//上面，k1和k2中的每一个都乘以各自的点。
//因为-k*p和k*p是一样的，而群定律是
//椭圆曲线表明p（x，y）=-p（x，-y），它更快，而且
//简化代码，使点为负。
	if signK1 == -1 {
		p1y, p1yNeg = p1yNeg, p1y
	}
	if signK2 == -1 {
		p2y, p2yNeg = p2yNeg, p2y
	}

//k1和k2的NAF版本应该有更多的零。
//
//
//包含-1。
	k1PosNAF, k1NegNAF := NAF(k1)
	k2PosNAF, k2NegNAF := NAF(k2)
	k1Len := len(k1PosNAF)
	k2Len := len(k2PosNAF)

	m := k1Len
	if m < k2Len {
		m = k2Len
	}

//使用NAF优化从左到右添加。见算法3.77
//来自[GECC ]。整体来说应该更快，因为会有很多
//
//以1倍的额外成本。
	var k1BytePos, k1ByteNeg, k2BytePos, k2ByteNeg byte
	for i := 0; i < m; i++ {
//既然我们从左到右，用0填充前面。
		if i < m-k1Len {
			k1BytePos = 0
			k1ByteNeg = 0
		} else {
			k1BytePos = k1PosNAF[i-(m-k1Len)]
			k1ByteNeg = k1NegNAF[i-(m-k1Len)]
		}
		if i < m-k2Len {
			k2BytePos = 0
			k2ByteNeg = 0
		} else {
			k2BytePos = k2PosNAF[i-(m-k2Len)]
			k2ByteNeg = k2NegNAF[i-(m-k2Len)]
		}

		for j := 7; j >= 0; j-- {
//q＝2＊q
			curve.doubleJacobian(qx, qy, qz, qx, qy, qz)

			if k1BytePos&0x80 == 0x80 {
				curve.addJacobian(qx, qy, qz, p1x, p1y, p1z,
					qx, qy, qz)
			} else if k1ByteNeg&0x80 == 0x80 {
				curve.addJacobian(qx, qy, qz, p1x, p1yNeg, p1z,
					qx, qy, qz)
			}

			if k2BytePos&0x80 == 0x80 {
				curve.addJacobian(qx, qy, qz, p2x, p2y, p2z,
					qx, qy, qz)
			} else if k2ByteNeg&0x80 == 0x80 {
				curve.addJacobian(qx, qy, qz, p2x, p2yNeg, p2z,
					qx, qy, qz)
			}
			k1BytePos <<= 1
			k1ByteNeg <<= 1
			k2BytePos <<= 1
			k2ByteNeg <<= 1
		}
	}

//将雅可比坐标场值转换回仿射big.ints。
	return curve.fieldJacobianToBigAffine(qx, qy, qz)
}

//scalarbasemult返回k*g，其中g是组的基点，k是
//
//椭圆曲线接口的一部分。
func (curve *KoblitzCurve) ScalarBaseMult(k []byte) (*big.Int, *big.Int) {
	newK := curve.moduloReduce(k)
	diff := len(curve.bytePoints) - len(newK)

//点Q=∞（无穷远点）。
	qx, qy, qz := new(fieldVal), new(fieldVal), new(fieldVal)

//对于每个8位窗口，curve.byte points都有256个字节点。这个
//策略是将字节点相加。这是最好的理解
//表示以256为基的k，它已经有点像了。
//8位窗口中的每个“数字”都可以使用字节点进行查找。
//加在一起。
	for i, byteVal := range newK {
		p := curve.bytePoints[diff+i][byteVal]
		curve.addJacobian(qx, qy, qz, &p[0], &p[1], &p[2], qx, qy, qz)
	}
	return curve.fieldJacobianToBigAffine(qx, qy, qz)
}

//qplus1div4返回用于计算的曲线的q+1/4常量
//平方根通过指数。
func (curve *KoblitzCurve) QPlus1Div4() *big.Int {
	return curve.q
}

var initonce sync.Once
var secp256k1 KoblitzCurve

func initAll() {
	initS256()
}

//FromHex将传递的十六进制字符串转换为大整数指针，并将
//恐慌是有错误。这仅适用于硬编码
//常量，以便可以检测到源代码中的错误。它只会（和
//只能）出于初始化目的调用。
func fromHex(s string) *big.Int {
	r, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic("invalid hex in source file: " + s)
	}
	return r
}

func initS256() {
//曲线参数取自[secg]第2.4.1节。
	secp256k1.CurveParams = new(elliptic.CurveParams)
	secp256k1.P = fromHex("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2F")
	secp256k1.N = fromHex("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141")
	secp256k1.B = fromHex("0000000000000000000000000000000000000000000000000000000000000007")
	secp256k1.Gx = fromHex("79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798")
	secp256k1.Gy = fromHex("483ADA7726A3C4655DA4FBFC0E1108A8FD17B448A68554199C47D08FFB10D4B8")
	secp256k1.BitSize = 256
	secp256k1.q = new(big.Int).Div(new(big.Int).Add(secp256k1.P,
		big.NewInt(1)), big.NewInt(4))
	secp256k1.H = 1
	secp256k1.halfOrder = new(big.Int).Rsh(secp256k1.N, 1)

//为方便起见，因为这将重复计算。
	secp256k1.byteSize = secp256k1.BitSize / 8

//反序列化并设置用于加速标量的预计算表
//基数乘法。这是硬编码数据，因此任何错误都是
//恐慌是因为它意味着源代码中有问题。
	if err := loadS256BytePoints(); err != nil {
		panic(err)
	}

//接下来的6个常量来自hal finney的bitcointalk.org帖子：
//https://bitcointalk.org/index.php？主题=3238.msg45565 msg45565
//愿他安息。
//
//它们还独立于
//gensecp256k1.go中的自同态向量函数。
	secp256k1.lambda = fromHex("5363AD4CC05C30E0A5261C028812645A122E22EA20816678DF02967C1B23BD72")
	secp256k1.beta = new(fieldVal).SetHex("7AE96A2B657C07106E64479EAC3434E99CF0497512F58995C1396C28719501EE")
	secp256k1.a1 = fromHex("3086D221A7D46BCDE86C90E49284EB15")
	secp256k1.b1 = fromHex("-E4437ED6010E88286F547FA90ABFE4C3")
	secp256k1.a2 = fromHex("114CA50F7A8E2F3F657C1108D9D44CFD8")
	secp256k1.b2 = fromHex("3086D221A7D46BCDE86C90E49284EB15")

//或者，我们可以使用下面的参数，但是，它们似乎
//大约慢8%。
//secp256k1.lambda=十六进制（“ac9c52b33fa3cf1f5ad9e3fd77ed9ba4a880b9fc8ec739c2e0cfc810b51283ce”）。
//secp256k1.beta=new（fieldval）.sethex（“851695d49A83f8ef919bb86153cbc16630fb68aed0a766a3ec693d68e6afa40”）
//secp256k1.a1=来自hex（“e4437ed6010e88286f547fa90abfe4c3”）
//secp256k1.b1=来自六角（“-3086d221a7d46bcde86c90e49284eb15”）
//secp256k1.a2=来自十六进制（“3086d221a7d46bcde86c90e49284eb15”）
//secp256k1.b2=来自六角（“114CA50F7A8E2F3F657C1108D9D44CFD8”）
}

//s256返回一条实现secp256k1的曲线。
func S256() *KoblitzCurve {
	initonce.Do(initAll)
	return &secp256k1
}
