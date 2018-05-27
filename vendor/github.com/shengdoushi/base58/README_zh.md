[![Travis CI](https://travis-ci.org/shengdoushi/base58.svg?branch=master)](https://travis-ci.org/shengdoushi/base58)
[![GoDoc](https://www.godoc.org/github.com/shengdoushi/base58?status.svg)](https://www.godoc.org/github.com/shengdoushi/base58)
[![Go Report Card](https://goreportcard.com/badge/github.com/shengdoushi/base58)](https://goreportcard.com/report/github.com/shengdoushi/base58)


## 特点

 * 快速轻量
 * API 语法简单
 * 内置常用的几种编码表: 比特币, IPFS, Flickr, Ripple
 * 可以自定义编码表
 * 自定义编码表可以是unicode字符串

## API Doc

[Godoc](https://www.godoc.org/github.com/shengdoushi/base58)

## base58 算法

类似base64编码算法， 但是去掉了几个看起来相同的字符(数字0, 大写字母O, 字母i的大写字母I, 字母L的小写字母l), 以及非字母数字字符(+,/).只含有字母，数字。优点是不易看错字符，且在大部分字符显示场景中，可以双击复制。

## 安装

```golang
go get -u github.com/shengdoushi/base58
```

## API

提供了 2 个API:

```
// 编码
func Encode(input []byte, alphabet *Alphabet)string

// 解码
func Decode(input string, alphabet *Alphabet)([]byte, error)
```

## 使用

```golang
import "github.com/shengdoushi/base58"
	
// 指定符号表
// myAlphabet := base58.BitcoinAlphabet // 使用 bitcoin 的符号表
myAlphabet := base58.NewAlphabet("ABCDEFGHJKLMNPQRSTUVWXYZ123456789abcdefghijkmnopqrstuvwxyz") // 自定义符号表
	
// 编码成 string 
var encodedStr string = base58.Encode([]byte{1,2,3,4}, myAlphabet)
	
// 解码为 []byte 
var encodedString string = "Xsdfjs123D"
decodedBytes, err := base58.Decode(encodedString, myAlphabet)
if err != nil {
	// error occurred
}
```

## 示例

```golang
package main

import (
	"fmt"
	"github.com/shengdoushi/base58"
)

func main(){
	// 这里使用比特币的符号表, 同 base58.NewAlphabet("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")
	myAlphabet := base58.BitcoinAlphabet
	
	// 编码
	input := []byte{0,0,0,1,2,3}
	var encodedString string = base58.Encode(input, myAlphabet)
	fmt.Printf("base58encode(%v) = %s\n", input, encodedString)
	
	// 解码， 如果输入的字符中有符号表中不含的字符会返回错误
	decodedBytes, err := base58.Decode(encodedString, myAlphabet)
	if err != nil {
		fmt.Println("error occurred: ", err)
	}else{
		fmt.Printf("base58decode(%s) = %v\n", encodedString, decodedBytes)
	}	
}
```

示例输出如下：

```
base58encode([0 0 0 1 2 3]) = 111Ldp
base58decode(111Ldp) = [0 0 0 1 2 3]
```

## 符号表

内置了几种常用的符号表(从 https://en.wikipedia.org/wiki/Base58 拷贝来)

```golang
// 比特币的符号表: 123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz
base58.BitcoinAlphabet
// IPFS的符号表: 123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz
base58.IPFSAlphabet
// Ripple的符号表: rpshnaf39wBUDNEGHJKLM4PQRST7VWXYZ2bcdeCg65jkm8oFqi1tuvAxyz
base58.RippleAlphabet
// Flickr的短链接使用的符号表: 123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ
base58.FlickrAlphabet
```

也可以自定义一个符号表， 注意符号表的长度(runes 的长度)必须是58， 否则会 panic， 可以指定为unicode字符串

```golang
myAlphabet := NewAlphabet("一二三四五六七八九十壹贰叁肆伍陆柒捌玖零拾佰仟万亿圆甲乙丙丁戊己庚辛壬癸子丑寅卯辰巳午未申酉戌亥金木水火土雷电风雨福")
```


## 协议

基于 MIT 协议, 查看 [LICENSE](LICENSE)。


