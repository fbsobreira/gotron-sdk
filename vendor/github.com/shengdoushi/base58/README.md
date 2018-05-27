[![Travis CI](https://travis-ci.org/shengdoushi/base58.svg?branch=master)](https://travis-ci.org/shengdoushi/base58)
[![GoDoc](https://www.godoc.org/github.com/shengdoushi/base58?status.svg)](https://www.godoc.org/github.com/shengdoushi/base58)
[![Go Report Card](https://goreportcard.com/badge/github.com/shengdoushi/base58)](https://goreportcard.com/report/github.com/shengdoushi/base58)

Chinese document 中文文档参看 [README_zh.md](README_zh.md)

## Features

 * Fast and lightweight
 * API simple
 * support some common alphabet: Bitcoin, IPFS, Flickr, Ripple
 * can custom alphabet
 * custom alphabet can be unicode string

## API Doc

[Godoc](https://www.godoc.org/github.com/shengdoushi/base58)

## base58 algorithm

Wikipedia:


From Wikipedia, the free encyclopedia
Base58 is a group of binary-to-text encoding schemes used to represent large integers as alphanumeric text. It is similar to Base64 but has been modified to avoid both non-alphanumeric characters and letters which might look ambiguous when printed. It is therefore designed for human users who manually enter the data, copying from some visual source, but also allows easy copy and paste because a double-click will usually select the whole string.

Compared to Base64, the following similar-looking letters are omitted: 0 (zero), O (capital o), I (capital i) and l (lower case L) as well as the non-alphanumeric characters + (plus) and / (slash). In contrast to Base64, the digits of the encoding do not line up well with byte boundaries of the original data. For this reason, the method is well-suited to encode large integers, but not designed to encode longer portions of binary data. The actual order of letters in the alphabet depends on the application, which is the reason why the term “Base58” alone is not enough to fully describe the format. A variant, Base56, excludes 1 (one) and o (lowercase o) compared to Base 58.

Base58Check is a Base58 encoding format that unambiguously encodes the type of data in the first few characters and includes an error detection code in the last few characters.[1]


## Installation

```golang
go get -u github.com/shengdoushi/base58
```

## API

just 2 API:

```
// encode with custom alphabet
func Encode(input []byte, alphabet *Alphabet)string

// Decode with custom alphabet
func Decode(input string, alphabet *Alphabet)([]byte, error)
```

## Usage

```golang
import "github.com/shengdoushi/base58"
	
// Alphabet
// myAlphabet := base58.BitcoinAlphabet // bitcoin address's alphabet
myAlphabet := base58.NewAlphabet("ABCDEFGHJKLMNPQRSTUVWXYZ123456789abcdefghijkmnopqrstuvwxyz") // custom alphabet, must 58 length
	
// encode to string 
var encodedStr string = base58.Encode([]byte{1,2,3,4}, myAlphabet)
	
// decode to []byte 
var encodedString string = "Xsdfjs123D"
decodedBytes, err := base58.Decode(encodedString, myAlphabet)
if err != nil {
	// error occurred
}
```

## Example

```golang
package main

import (
	"fmt"
	"github.com/shengdoushi/base58"
)

func main(){
	// use bitcoin alphabet, just same as: base58.NewAlphabet("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")
	myAlphabet := base58.BitcoinAlphabet
	
	// encode
	input := []byte{0,0,0,1,2,3}
	var encodedString string = base58.Encode(input, myAlphabet)
	fmt.Printf("base58encode(%v) = %s\n", input, encodedString)
	
	// decode
	decodedBytes, err := base58.Decode(encodedString, myAlphabet)
	if err != nil { // error occurred when encodedString contains character not in alphabet
		fmt.Println("error occurred: ", err)
	}else{
		fmt.Printf("base58decode(%s) = %v\n", encodedString, decodedBytes)
	}	
}
```


The example will output:

```
base58encode([0 0 0 1 2 3]) = 111Ldp
base58decode(111Ldp) = [0 0 0 1 2 3]
```

## Alphabet

This package provide some common alphabet(copyed from https://en.wikipedia.org/wiki/Base58):

```golang
// Bitcoin: 123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz
base58.BitcoinAlphabet
// IPFS: 123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz
base58.IPFSAlphabet
// Ripple: rpshnaf39wBUDNEGHJKLM4PQRST7VWXYZ2bcdeCg65jkm8oFqi1tuvAxyz
base58.RippleAlphabet
// Flickr: 123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ
base58.FlickrAlphabet
```

Or you can use custom alphabet, the alphabet's length (runes's length)  must be 58. The alphabet can use unicode string.

```golang
myAlphabet := base58.NewAlphabet("一二三四五六七八九十壹贰叁肆伍陆柒捌玖零拾佰仟万亿圆甲乙丙丁戊己庚辛壬癸子丑寅卯辰巳午未申酉戌亥金木水火土雷电风雨福")
```


## LICENSE

Under the  MIT LICENSE, see the [LICENSE](LICENSE) file for details.


