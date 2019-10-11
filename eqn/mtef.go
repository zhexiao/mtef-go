package eqn

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"fmt"
	"github.com/extrame/ole2"
	"io"
	"log"
)

const oleCbHdr = uint16(28)

// [MTEFv5](https://docs.wiris.com/en/mathtype/mathtype_desktop/mathtype-sdk/mtef5)
type MTEFv5 struct {
	mMtefVer     uint8
	mPlatform    uint8
	mProduct     uint8
	mVersion     uint8
	mVersionSub  uint8
	mApplication string
	mInline      uint8

	reader io.ReadSeeker

	ast   *MtAST
	nodes []*MtAST
}

func (m *MTEFv5) readRecord() (err error) {
	/**
	读取body的每一行数据并保存到数组里
	 */

	//Header
	_ = binary.Read(m.reader, binary.LittleEndian, &m.mMtefVer)
	_ = binary.Read(m.reader, binary.LittleEndian, &m.mPlatform)
	_ = binary.Read(m.reader, binary.LittleEndian, &m.mProduct)
	_ = binary.Read(m.reader, binary.LittleEndian, &m.mVersion)
	_ = binary.Read(m.reader, binary.LittleEndian, &m.mVersionSub)
	m.mApplication, _ = m.readNullTerminatedString()
	_ = binary.Read(m.reader, binary.LittleEndian, &m.mInline)

	//fmt.Println(m.mMtefVer, m.mPlatform, m.mProduct, m.mVersion, m.mVersionSub)
	//fmt.Println(m.mInline)

	//Body
	for {
		record := RecordType(0)
		err = binary.Read(m.reader, binary.LittleEndian, &record)

		// 根据future定义，>=100的后面会跟一个字节，这个字节代表需要跳过的长度
		//For now, readers can assume that an unsigned integer follows the record type and is the number of bytes following it in the record
		//This makes it easy for software that reads MTEF to skip these records.
		if record >= FUTURE {
			var skipFutureLength uint8
			_ = binary.Read(m.reader, binary.LittleEndian, &skipFutureLength)

			_, _ = m.reader.Seek(int64(skipFutureLength), io.SeekCurrent)
			continue
		}

		//debug 使用
		//fmt.Println(record)

		if err != nil {
			break
		}
		switch record {
		case END:
			m.nodes = append(m.nodes, &MtAST{END, nil, nil})
		case LINE:
			line := new(MtLine)
			_ = m.readLine(line)

			m.nodes = append(m.nodes, &MtAST{LINE, line, nil})
		case CHAR:
			char := new(MtChar)
			_ = m.readChar(char)

			m.nodes = append(m.nodes, &MtAST{CHAR, char, nil})
		case TMPL:
			tmpl := new(MtTmpl)
			_ = m.readTMPL(tmpl)

			m.nodes = append(m.nodes, &MtAST{TMPL, tmpl, nil})
		case PILE:
			pile := new(MtPile)
			_ = m.readPile(pile)

			m.nodes = append(m.nodes, &MtAST{PILE, pile, nil})
		case MATRIX:
			matrix := new(MtMatrix)
			_ = m.readMatrix(matrix)

			m.nodes = append(m.nodes, &MtAST{MATRIX, matrix, nil})

			//匹配矩阵数据下面的2个nil
			m.nodes = append(m.nodes, &MtAST{LINE, new(MtLine), nil})
			m.nodes = append(m.nodes, &MtAST{LINE, new(MtLine), nil})
		case EMBELL:
			embell := new(MtEmbellRd)
			_ = m.readEmbell(embell)

			m.nodes = append(m.nodes, &MtAST{tag: EMBELL, value: embell, children: nil})
		case FONT_STYLE_DEF:
			fsDef := new(MtfontStyleDef)
			_ = binary.Read(m.reader, binary.LittleEndian, &fsDef.fontDefIndex)
			fsDef.name, _ = m.readNullTerminatedString()

			//读取字节，但是不关心数据，注释
			//m.nodes = append(m.nodes, &MtAST{FONT_STYLE_DEF, fsDef, nil})
		case SUB:
			m.nodes = append(m.nodes, &MtAST{SUB, nil, nil})
		case SUB2:
			m.nodes = append(m.nodes, &MtAST{SUB2, nil, nil})
		case SYM:
			m.nodes = append(m.nodes, &MtAST{SYM, nil, nil})
		case SUBSYM:
			m.nodes = append(m.nodes, &MtAST{SUBSYM, nil, nil})
		case FONT_DEF:
			fdef := new(MtfontDef)
			_ = binary.Read(m.reader, binary.LittleEndian, &fdef.encDefIndex)
			fdef.name, _ = m.readNullTerminatedString()

			m.nodes = append(m.nodes, &MtAST{FONT_DEF, fdef, nil})
		case COLOR:
			cIndex := new(MtColorDefIndex)
			_ = binary.Read(m.reader, binary.LittleEndian, &cIndex.index)

			//读取字节，但是不关心数据，注释
			//m.nodes = append(m.nodes, &MtAST{tag: COLOR, value: cIndex, children: nil})
		case COLOR_DEF:
			cDef := new(MtColorDef)
			_ = m.readColorDef(cDef)

			//读取字节，但是不关心数据，注释
			//m.nodes = append(m.nodes, &MtAST{tag: COLOR_DEF, value: cDef, children: nil})
		case FULL:
			m.nodes = append(m.nodes, &MtAST{FULL, nil, nil})
		case EQN_PREFS:
			prefs := new(MtEqnPrefs)
			_ = m.readEqnPrefs(prefs)

			m.nodes = append(m.nodes, &MtAST{EQN_PREFS, prefs, nil})
		case ENCODING_DEF:
			enc, _ := m.readNullTerminatedString()

			m.nodes = append(m.nodes, &MtAST{ENCODING_DEF, enc, nil})
		default:
			fmt.Println("FUTURE RECORD", record)
		}
	}

	return nil
}

func (m *MTEFv5) readNullTerminatedString() (s string, err error) {
	buf, p := bytes.Buffer{}, []byte{0}
	for {
		_, err = m.reader.Read(p)
		if p[0] == 0 {
			break
		}
		buf.WriteByte(p[0])
	}
	return buf.String(), err
}

func (m *MTEFv5) readLine(line *MtLine) (err error) {
	options := OptionType(0)
	err = binary.Read(m.reader, binary.LittleEndian, &options)

	if MtefOptNudge == MtefOptNudge&options {
		line.nudgeX, line.nudgeY, _ = m.readNudge()
	}
	if MtefOptLineLspace == MtefOptLineLspace&options {
		_ = binary.Read(m.reader, binary.LittleEndian, &line.lineSpace)
	}

	//RULER解析
	if mtefOPT_LP_RULER == mtefOPT_LP_RULER&options {
		var nStops uint8
		_ = binary.Read(m.reader, binary.LittleEndian, &nStops)

		var tabList []uint8
		for i := uint8(0); i < nStops; i++ {
			var stopVal uint8
			_ = binary.Read(m.reader, binary.LittleEndian, &stopVal)
			tabList = append(tabList, stopVal)

			var tabOffset uint16
			_ = binary.Read(m.reader, binary.LittleEndian, &tabOffset)
		}
	}

	if MtefOptLineNull == MtefOptLineNull&options {
		line.null = true
	}

	return err
}

func (m *MTEFv5) readDimensionArrays(size int64) (array []string, err error) {
	var flag = true
	var tmpStr = new(bytes.Buffer)
	var count = int64(0)

	var fx = func(x uint8) {
		if flag {
			switch x {
			case 0x00:
				flag = false
				tmpStr.WriteString("in")
			case 0x01:
				flag = false
				tmpStr.WriteString("cm")
			case 0x02:
				flag = false
				tmpStr.WriteString("pt")
			case 0x03:
				flag = false
				tmpStr.WriteString("pc")
			case 0x04:
				flag = false
				tmpStr.WriteString("%")
			default:
				fmt.Println("invalid bytes")
			}
		} else {
			switch x {
			case 0x00:
				flag = false
				tmpStr.WriteByte('0')
			case 0x01:
				flag = false
				tmpStr.WriteByte('1')
			case 0x02:
				flag = false
				tmpStr.WriteByte('2')
			case 0x03:
				flag = false
				tmpStr.WriteByte('3')
			case 0x04:
				flag = false
				tmpStr.WriteByte('4')
			case 0x05:
				flag = false
				tmpStr.WriteByte('5')
			case 0x06:
				flag = false
				tmpStr.WriteByte('6')
			case 0x07:
				flag = false
				tmpStr.WriteByte('7')
			case 0x08:
				flag = false
				tmpStr.WriteByte('8')
			case 0x09:
				flag = false
				tmpStr.WriteByte('9')
			case 0x0a:
				flag = false
				tmpStr.WriteByte('.')
			case 0x0b:
				flag = false
				tmpStr.WriteByte('-')
			case 0x0f:
				flag = true
				count += 1
				array = append(array, tmpStr.String())
				tmpStr.Reset()
			default:
				fmt.Println("invalid bytes")
			}
		}
	}

	for {
		if count >= size {
			//fmt.Println("break with size=", size)
			break
		}
		ch := uint8(0)
		_ = binary.Read(m.reader, binary.LittleEndian, &ch)

		//fmt.Println("ch=", ch)

		hi := (ch & 0xf0) / 16
		lo := ch & 0x0f
		fx(hi)
		fx(lo)
	}
	return array, nil
}

func (m *MTEFv5) readEqnPrefs(eqnPrefs *MtEqnPrefs) (err error) {
	options := uint8(0)
	_ = binary.Read(m.reader, binary.LittleEndian, &options)

	//sizes
	size := uint8(0)
	_ = binary.Read(m.reader, binary.LittleEndian, &size)
	eqnPrefs.sizes, _ = m.readDimensionArrays(int64(size))

	//spaces
	size = 0
	_ = binary.Read(m.reader, binary.LittleEndian, &size)
	eqnPrefs.spaces, _ = m.readDimensionArrays(int64(size))

	//styles
	size = 0
	_ = binary.Read(m.reader, binary.LittleEndian, &size)
	styles := make([]byte, size)
	for i := uint8(0); i < size; i ++ {
		c := uint8(0)
		_ = binary.Read(m.reader, binary.LittleEndian, &c)
		if c == 0 {
			styles = append(styles, 0)
		} else {
			_ = binary.Read(m.reader, binary.LittleEndian, &c)
			styles = append(styles, c)
		}
	}
	eqnPrefs.styles = styles
	return nil
}

func (m *MTEFv5) readChar(char *MtChar) (err error) {
	options := OptionType(0)
	_ = binary.Read(m.reader, binary.LittleEndian, &options)

	if MtefOptNudge == MtefOptNudge&options {
		char.nudgeX, char.nudgeY, _ = m.readNudge()
	}
	_ = binary.Read(m.reader, binary.LittleEndian, &char.typeface)

	if MtefOptCharEncNoMtcode != MtefOptCharEncNoMtcode&options {
		_ = binary.Read(m.reader, binary.LittleEndian, &char.mtcode)
	}
	if MtefOptCharEncChar8 == MtefOptCharEncChar8&options {
		_ = binary.Read(m.reader, binary.LittleEndian, &char.bits8)
	}
	if MtefOptCharEncChar16 == MtefOptCharEncChar16&options {
		_ = binary.Read(m.reader, binary.LittleEndian, &char.bits16)

	}
	return nil
}

func (m *MTEFv5) readNudge() (nudgeX int16, nudgeY int16, err error) {
	b1 := 0
	b2 := 0
	_ = binary.Read(m.reader, binary.LittleEndian, &b1)
	_ = binary.Read(m.reader, binary.LittleEndian, &b2)

	if b1 == 128 || b2 == 128 {
		_ = binary.Read(m.reader, binary.LittleEndian, &nudgeX)
		_ = binary.Read(m.reader, binary.LittleEndian, &nudgeY)
		return nudgeX, nudgeY, err
	} else {
		nudgeX = int16(b1)
		nudgeY = int16(b2)
		return nudgeX, nudgeY, err
	}
}

func (m *MTEFv5) readTMPL(tmpl *MtTmpl) (err error) {
	options := OptionType(0)
	_ = binary.Read(m.reader, binary.LittleEndian, &options)

	if MtefOptNudge == MtefOptNudge&options {
		tmpl.nudgeX, tmpl.nudgeY, _ = m.readNudge()
	}
	_ = binary.Read(m.reader, binary.LittleEndian, &tmpl.selector)

	// variation, 1 or 2 bytes
	byte1 := uint8(0)
	_ = binary.Read(m.reader, binary.LittleEndian, &byte1)
	if 0x80 == byte1&0x80 {
		byte2 := uint8(0)
		_ = binary.Read(m.reader, binary.LittleEndian, &byte2)
		tmpl.variation = (uint16(byte1) & 0x7F) | (uint16(byte2) << 8)
	} else {
		tmpl.variation = uint16(byte1)
	}
	_ = binary.Read(m.reader, binary.LittleEndian, &tmpl.options)
	return nil
}

func (m *MTEFv5) readPile(pile *MtPile) (err error) {
	options := OptionType(0)
	_ = binary.Read(m.reader, binary.LittleEndian, &options)

	if MtefOptNudge == MtefOptNudge&options {
		pile.nudgeX, pile.nudgeY, _ = m.readNudge()
	}

	//读取halign和valign
	_ = binary.Read(m.reader, binary.LittleEndian, &pile.halign)
	_ = binary.Read(m.reader, binary.LittleEndian, &pile.valign)

	return nil
}

func (m *MTEFv5) readMatrix(matrix *MtMatrix) (err error) {
	options := OptionType(0)
	_ = binary.Read(m.reader, binary.LittleEndian, &options)

	if MtefOptNudge == MtefOptNudge&options {
		matrix.nudgeX, matrix.nudgeY, _ = m.readNudge()
	}

	//读取valign和h_just、v_just
	_ = binary.Read(m.reader, binary.LittleEndian, &matrix.valign)
	_ = binary.Read(m.reader, binary.LittleEndian, &matrix.h_just)
	_ = binary.Read(m.reader, binary.LittleEndian, &matrix.v_just)

	//读取rows和cols
	_ = binary.Read(m.reader, binary.LittleEndian, &matrix.rows)
	_ = binary.Read(m.reader, binary.LittleEndian, &matrix.cols)

	//fmt.Printf("%v", matrix)
	return nil
}

func (m *MTEFv5) readEmbell(embell *MtEmbellRd) (err error) {
	options := OptionType(0)
	_ = binary.Read(m.reader, binary.LittleEndian, &options)

	if MtefOptNudge == MtefOptNudge&options {
		embell.nudgeX, embell.nudgeY, _ = m.readNudge()
	}

	//读取embellishment type
	_ = binary.Read(m.reader, binary.LittleEndian, &embell.embellType)
	return nil
}

func (m *MTEFv5) readColorDef(colorDef *MtColorDef) (err error) {
	options := OptionType(0)
	_ = binary.Read(m.reader, binary.LittleEndian, &options)

	var color uint16
	if mtefCOLOR_CMYK == mtefCOLOR_CMYK&options {
		//CMYK，读4个值
		for i := 0; i < 4; i++ {
			_ = binary.Read(m.reader, binary.LittleEndian, &color)
			colorDef.values = append(colorDef.values, uint8(color))
		}
	} else {
		//	RGB，读3个值
		for i := 0; i < 3; i++ {
			_ = binary.Read(m.reader, binary.LittleEndian, &color)
			colorDef.values = append(colorDef.values, uint8(color))
		}
	}

	if mtefCOLOR_NAME == mtefCOLOR_NAME&options {
		colorDef.name, _ = m.readNullTerminatedString()
	}

	return nil
}

func (m *MTEFv5) Translate() (latex string, err error) {
	latexStr, err := m.makeLatex(m.ast)
	if err != nil {
		fmt.Println(err)
	}
	return latexStr, nil
}

func (m *MTEFv5) makeAST() (err error) {
	/**
	根据数组生成出栈入栈结构
	 */
	ast := new(MtAST)
	ast.tag = 0xff
	ast.value = nil
	m.ast = ast

	stack := list.New()
	stack.PushBack(ast)

	for _, node := range m.nodes {
		//debug 可用
		//fmt.Printf("%+v %+v \n", node.tag, node.value)

		switch node.tag {
		case LINE:
			if stack.Len() > 0 {
				ele := stack.Back()

				//将对象强制转为MtAST类型
				parent := ele.Value.(*MtAST)

				parent.children = append(parent.children, node)
			}
			if !node.value.(*MtLine).null {
				//如果与0 <nil> 匹配，则需要入栈
				stack.PushBack(node)
			}
		case TMPL:
			if stack.Len() > 0 {
				ele := stack.Back()

				//将对象强制转为MtAST类型
				parent := ele.Value.(*MtAST)

				parent.children = append(parent.children, node)
			}

			//如果与0 <nil> 匹配，则需要入栈
			stack.PushBack(node)
		case PILE:
			if stack.Len() > 0 {
				ele := stack.Back()

				//将对象强制转为MtAST类型
				parent := ele.Value.(*MtAST)

				parent.children = append(parent.children, node)
			}

			//如果与0 <nil> 匹配，则需要入栈
			stack.PushBack(node)
		case MATRIX:
			if stack.Len() > 0 {
				ele := stack.Back()

				//将对象强制转为MtAST类型
				parent := ele.Value.(*MtAST)

				parent.children = append(parent.children, node)
			}

			//如果与0 <nil> 匹配，则需要入栈
			stack.PushBack(node)
		case END:
			if stack.Len() > 0 {
				ele := stack.Back()
				stack.Remove(ele)
			}
		case CHAR:
			if stack.Len() > 0 {
				ele := stack.Back()

				//将对象强制转为MtAST类型
				parent := ele.Value.(*MtAST)

				parent.children = append(parent.children, node)
			} else if stack.Len() == 0 {
				//never go there
				ast.children = append(ast.children, node)
			}
		case EMBELL:
			if stack.Len() > 0 {
				//读取父节点
				ele := stack.Back()

				//并将对象强制转为MtAST类型
				parent := ele.Value.(*MtAST)
				parent.children = append(parent.children, node)

				switch EmbellType(node.value.(*MtEmbellRd).embellType) {
				//数据结构中，这些数据是在字符后面，但是在latex展示中某些字符需要在字符前面
				//比如： $$ \hat y $$
				//所以我们需要交换最后2位
				case emb1DOT, embHAT, embOBAR:
					if len(parent.children) >= 2 {
						embellData := parent.children[len(parent.children)-1]
						charData := parent.children[len(parent.children)-2]
						parent.children = parent.children[:len(parent.children)-2]

						parent.children = append(parent.children, embellData, charData)
					}
				}
			}

			//如果与0 <nil> 匹配，则需要入栈
			stack.PushBack(node)

			//case COLOR_DEF:
			//	/*
			//	这个数据结构有3或4个（RGB或者CMYK）对应的nil，所以需要循环把每个值都push到栈里面
			//
			//	16 &{values:[0 0 0] name:}
			//	0 <nil>
			//	0 <nil>
			//	0 <nil>
			//	 */
			//
			//	colorList := node.value.(*MtColorDef).values
			//	if len(colorList) > 0 {
			//		//读取每个color的值，然后入栈
			//		for _, val := range colorList {
			//			//如果与0 <nil> 匹配，则需要入栈
			//			stack.PushBack(val)
			//		}
			//	}
			//case FONT_STYLE_DEF:
			//	/*
			//	这个数据结构如下，所以需要配对6个入栈
			//	8 &{fontDefIndex:1 name:}
			//	0 <nil>
			//	0 <nil>
			//	0 <nil>
			//	0 <nil>
			//	0 <nil>
			//	0 <nil>
			//	*/
			//
			//	fontIndex := node.value.(*MtfontStyleDef).fontDefIndex
			//	if fontIndex == 1 {
			//		for i := 0; i < 6; i++ {
			//			//如果与0 <nil> 匹配，则需要入栈
			//			stack.PushBack(0)
			//		}
			//	}
		}
	}

	//m.ast.debug(0)
	return nil
}

func (m *MTEFv5) makeLatex(ast *MtAST) (latex string, err error) {
	/**
	根据出栈入栈结构生成latex字符串
	 */

	buf := new(bytes.Buffer)

	switch ast.tag {
	case ROOT:
		buf.WriteString("$$ ")
		for _, _ast := range ast.children {
			_latex, _ := m.makeLatex(_ast)
			buf.WriteString(_latex)
		}
		buf.WriteString(" $$")
		return buf.String(), nil
	case CHAR:
		mtcode := ast.value.(*MtChar).mtcode
		typeface := ast.value.(*MtChar).typeface
		char := string(mtcode)

		//生成char的一些特殊集
		hexExtend := ""
		typefaceFmt := ""
		switch typeface - 128 {
		case fnMTEXTRA:
			hexExtend = "/mathmode"
		case fnSPACE:
			hexExtend = "/mathmode"
		case fnTEXT:
			typefaceFmt = "{ \\rm{ %v } }"
		}

		//生成扩展字符的key
		hexCode := fmt.Sprintf("%04x", mtcode)
		hexKey := fmt.Sprintf("char/0x%v%v", hexCode, hexExtend)

		//fmt.Println(char, hexKey)

		//首先去找扩展字符
		sChar, ok := Chars[hexKey]
		if ok {
			char = sChar
		} else {
			//如果char是特殊symbol，需要转义
			sChar, ok := SpecialChar[char]
			if ok {
				char = sChar
			}
		}

		//确定字符是否为文本，如果是文本，则需要包一层
		if typefaceFmt != "" {
			char = fmt.Sprintf(typefaceFmt, char)
		}

		buf.WriteString(char)
		return buf.String(), nil
	case TMPL:
		//强制类型转换为MtTmpl
		tmpl := ast.value.(*MtTmpl)

		switch SelectorType(tmpl.selector) {
		case tmANGLE:
			mainAST := ast.children[0]
			leftAST := ast.children[1]
			rightAST := ast.children[2]

			mainSlot, _ := m.makeLatex(mainAST)
			leftSlot, _ := m.makeLatex(leftAST)
			rightSlot, _ := m.makeLatex(rightAST)

			//转成latex代码
			var mainStr, leftStr, rightStr string
			if mainSlot != "" {
				mainStr = fmt.Sprintf("{ %v }", mainSlot)
			}
			if leftSlot != "" {
				leftStr = fmt.Sprintf("\\left %v", leftSlot)
			}
			if rightSlot != "" {
				rightStr = fmt.Sprintf("\\right %v", rightSlot)
			}

			buf.WriteString(fmt.Sprintf("%v %v %v", leftStr, mainStr, rightStr))
			return buf.String(), nil

		case tmPAREN:
			mainAST := ast.children[0]
			leftAST := ast.children[1]
			rightAST := ast.children[2]

			mainSlot, _ := m.makeLatex(mainAST)
			leftSlot, _ := m.makeLatex(leftAST)
			rightSlot, _ := m.makeLatex(rightAST)

			//转成latex代码
			var mainStr, leftStr, rightStr string
			if mainSlot != "" {
				mainStr = fmt.Sprintf("{ %v }", mainSlot)
			}
			if leftSlot != "" {
				leftStr = fmt.Sprintf("\\left %v", leftSlot)
			}
			if rightSlot != "" {
				rightStr = fmt.Sprintf("\\right %v", rightSlot)
			}

			buf.WriteString(fmt.Sprintf("%v %v %v", leftStr, mainStr, rightStr))
			return buf.String(), nil
		case tmBRACE:
			var mainSlot, leftSlot, rightSlot string
			for idx, astData := range ast.children {
				if idx == 0 {
					mainSlot, _ = m.makeLatex(astData)
				} else if idx == 1 {
					leftSlot, _ = m.makeLatex(astData)
				} else {
					rightSlot, _ = m.makeLatex(astData)
				}
			}

			if rightSlot == "" {
				rightSlot = "."
			} else {
				rightSlot = " " + rightSlot
			}

			//组装公式
			buf.WriteString(fmt.Sprintf(
				"\\left %v \\begin{array}{l} %v \\end{array} \\right%v",
				leftSlot, mainSlot, rightSlot))

			return buf.String(), nil
		case tmBRACK:
			mainAST := ast.children[0]
			leftAST := ast.children[1]
			rightAST := ast.children[2]
			mainSlot, _ := m.makeLatex(mainAST)
			if mainSlot == "" {
				mainSlot = "\\space"
			}
			leftSlot, _ := m.makeLatex(leftAST)
			rightSlot, _ := m.makeLatex(rightAST)
			buf.WriteString(fmt.Sprintf("\\left%v %v \\right%v", leftSlot, mainSlot, rightSlot))
			return buf.String(), nil
		case tmBAR:
			//读取数据 ParBoxClass
			var mainSlot, leftSlot, rightSlot string
			for idx, astData := range ast.children {
				if idx == 0 {
					mainSlot, _ = m.makeLatex(astData)
				} else if idx == 1 {
					leftSlot, _ = m.makeLatex(astData)
				} else {
					rightSlot, _ = m.makeLatex(astData)
				}
			}

			if rightSlot == "" {
				rightSlot = "."
			} else {
				rightSlot = " " + rightSlot
			}

			//转成latex代码
			var mainStr, leftStr, rightStr string
			if mainSlot != "" {
				mainStr = fmt.Sprintf("{ %v }", mainSlot)
			}
			if leftSlot != "" {
				leftStr = fmt.Sprintf("\\left %v", leftSlot)
			}
			if rightSlot != "" {
				rightStr = fmt.Sprintf("\\right %v", rightSlot)
			}

			//组成整体公式
			tmplStr := fmt.Sprintf("%v %v %v", leftStr, mainStr, rightStr)
			buf.WriteString(tmplStr)

			return buf.String(), nil
		case tmINTERVAL:
			//读取数据 ParBoxClass
			mainAST := ast.children[0]
			leftAST := ast.children[1]
			rightAST := ast.children[2]

			//读取latex数据
			mainSlot, _ := m.makeLatex(mainAST)
			leftSlot, _ := m.makeLatex(leftAST)
			rightSlot, _ := m.makeLatex(rightAST)

			//转成latex代码
			var mainStr, leftStr, rightStr string
			if mainSlot != "" {
				mainStr = fmt.Sprintf("{ %v }", mainSlot)
			}
			if leftSlot != "" {
				leftStr = fmt.Sprintf("\\left %v", leftSlot)
			}
			if rightSlot != "" {
				rightStr = fmt.Sprintf("\\right %v", rightSlot)
			}

			//组成整体公式
			tmplStr := fmt.Sprintf("%v %v %v", leftStr, mainStr, rightStr)
			buf.WriteString(tmplStr)

			return buf.String(), nil
		case tmROOT:
			mainAST := ast.children[0]
			radiAST := ast.children[1]
			mainSlot, _ := m.makeLatex(mainAST)
			radiSlot, _ := m.makeLatex(radiAST)
			buf.WriteString(fmt.Sprintf("\\sqrt[%v] { %v }", radiSlot, mainSlot))
			return buf.String(), nil
		case tmFRACT:
			numAST := ast.children[0]
			denAST := ast.children[1]
			numSlot, _ := m.makeLatex(numAST)
			denSlot, _ := m.makeLatex(denAST)
			buf.WriteString(fmt.Sprintf("\\frac { %v } { %v }", numSlot, denSlot))
			return buf.String(), nil
		case tmARROW:
			/*
			variation	symbol	description
			0×0000	tvAR_SINGLE	single arrow
			0×0001	tvAR_DOUBLE	double arrow
			0×0002	tvAR_HARPOON	harpoon
			0×0004	tvAR_TOP	top slot is present
			0×0008	tvAR_BOTTOM	bottom slot is present
			0×0010	tvAR_LEFT	if single, arrow points left
			0×0020	tvAR_RIGHT	if single, arrow points right
			0×0010	tvAR_LOS	if double or harpoon, large over small
			0×0020	tvAR_SOL	if double or harpoon, small over large
			 */
			topAST := ast.children[0]
			bottomAST := ast.children[1]

			//读取latex数据
			topSlot, _ := m.makeLatex(topAST)
			bottomSlot, _ := m.makeLatex(bottomAST)

			//转成latex代码
			var topStr, bottomStr string
			if topSlot != "" {
				topStr = fmt.Sprintf("{\\mathrm{ %v }}", topSlot)
			}
			if bottomSlot != "" {
				bottomStr = fmt.Sprintf("[\\mathrm{ %v }]", bottomSlot)
			}

			/*
			variation转码
			 */
			variationsMap := make(map[uint16]string)
			variationsMap[0x0000] = "single"
			variationsMap[0x0001] = "double"
			variationsMap[0x0002] = "harpoon"
			variationsMap[0x0004] = "topSlotPresent"
			variationsMap[0x0008] = "bottomSlotPresent"
			variationsMap[0x0010] = "pointLeft"
			variationsMap[0x0020] = "pointRight"

			//有序循环
			variationsCode := []uint16{0x0000, 0x0001, 0x0002, 0x0004, 0x0008, 0x0010, 0x0020}

			arrowStyle := "single"
			latexFmt := "\\x"
			for _, vCode := range variationsCode {
				//如果存在掩码
				if vCode&uint16(tmpl.variation) != 0 {
					//判断类型，默认是single
					if variationsMap[vCode] == "double" {
						arrowStyle = "double"
					} else if variationsMap[vCode] == "harpoon" {
						arrowStyle = "harpoon"
					}

					if arrowStyle == "single" && variationsMap[vCode] == "pointLeft" {
						latexFmt = latexFmt + "leftarrow"
					} else if arrowStyle == "double" && variationsMap[vCode] == "pointLeft" {
						fmt.Println("not implement double , large over small")
					} else if arrowStyle == "harpoon" && variationsMap[vCode] == "pointLeft" {
						fmt.Println("not implement harpoon, large over small")
					}

					if arrowStyle == "single" && variationsMap[vCode] == "pointRight" {
						latexFmt = latexFmt + "rightarrow"
					} else if arrowStyle == "double" && variationsMap[vCode] == "pointRight" {
						fmt.Println("not implement double , small over large")
					} else if arrowStyle == "harpoon" && variationsMap[vCode] == "pointRight" {
						fmt.Println("not implement harpoon, small over large")
					}
				}
			}
			/*
			variation转码 END
			 */

			//组成整体公式
			tmplStr := fmt.Sprintf("%v %v %v", latexFmt, bottomStr, topStr)
			buf.WriteString(tmplStr)

			return buf.String(), nil
		case tmUBAR:
			//读取数据
			mainAST := ast.children[0]

			//读取latex数据
			mainSlot, _ := m.makeLatex(mainAST)

			//转成latex代码
			var mainStr string
			if mainSlot != "" {
				mainStr = fmt.Sprintf(" {\\underline{ %v }} ", mainSlot)
			}

			//组成整体公式
			tmplStr := fmt.Sprintf(" %v ", mainStr)
			buf.WriteString(tmplStr)

			//返回数据
			return buf.String(), nil
		case tmSUM:
			//读取数据 BigOpBoxClass
			var mainSlot, upperSlot, lowerSlot, operatorSlot string
			for idx, astData := range ast.children {
				if idx == 0 {
					mainSlot, _ = m.makeLatex(astData)
				} else if idx == 1 {
					lowerSlot, _ = m.makeLatex(astData)
				} else if idx == 2 {
					upperSlot, _ = m.makeLatex(astData)
				} else {
					operatorSlot, _ = m.makeLatex(astData)
				}
			}

			//转成latex代码
			var mainStr, lowerStr, upperStr string
			if mainSlot != "" {
				mainStr = fmt.Sprintf("{ %v }", mainSlot)
			}
			if lowerSlot != "" {
				lowerStr = fmt.Sprintf("\\limits_{ %v }", lowerSlot)
			}
			if upperSlot != "" {
				upperStr = fmt.Sprintf("^ %v", upperSlot)
			}

			//组成整体公式
			tmplStr := fmt.Sprintf("%v %v %v %v", operatorSlot, lowerStr, upperStr, mainStr)
			buf.WriteString(tmplStr)

			return buf.String(), nil
		case tmLIM:
			//读取数据 LimBoxClass
			var mainSlot, lowerSlot, upperSlot string
			for idx, astData := range ast.children {
				if idx == 0 {
					mainSlot, _ = m.makeLatex(astData)
				} else if idx == 1 {
					lowerSlot, _ = m.makeLatex(astData)
				} else {
					upperSlot, _ = m.makeLatex(astData)
				}
			}

			//转成latex代码
			var mainStr, lowerStr, upperStr string
			if mainSlot != "" {
				mainStr = fmt.Sprintf("\\mathop { %v }", mainSlot)
			}
			if lowerSlot != "" {
				lowerStr = fmt.Sprintf("\\limits_{ %v }", lowerSlot)
			}
			if upperSlot != "" {
				upperStr = ""
			}

			//组成整体公式
			tmplStr := fmt.Sprintf("%v %v %v", mainStr, lowerStr, upperStr)
			buf.WriteString(tmplStr)

			return buf.String(), nil
		case tmSUP:
			subAST := ast.children[0]
			supAST := ast.children[1]
			subSlot, _ := m.makeLatex(subAST)
			supSlot, _ := m.makeLatex(supAST)

			buf.WriteString(" ^ { ")
			buf.WriteString(supSlot)
			buf.WriteString(" } ")
			if subSlot != "" {
				buf.WriteString(" { ")
				buf.WriteString(subSlot)
				buf.WriteString(" } ")
			}
			return buf.String(), nil
		case tmSUB:
			//读取下标和上标
			subAST := ast.children[0]
			supAST := ast.children[1]

			//读取latex数据
			subSlot, _ := m.makeLatex(subAST)
			supSlot, _ := m.makeLatex(supAST)

			//转成latex代码
			var subFmt, supFmt string
			if subSlot != "" {
				subFmt = fmt.Sprintf("_{ %v }", subSlot)
			}
			if supSlot != "" {
				supFmt = fmt.Sprintf("^{ %v }", supSlot)
			}

			//组成整体公式
			tmplStr := fmt.Sprintf("%v  %v", subFmt, supFmt)
			buf.WriteString(tmplStr)

			//返回数据
			return buf.String(), nil
		case tmSUBSUP:
			//读取下标和上标
			subAST := ast.children[0]
			supAST := ast.children[1]

			//读取latex数据
			subSlot, _ := m.makeLatex(subAST)
			supSlot, _ := m.makeLatex(supAST)

			//转成latex代码
			var subFmt, supFmt string
			if subSlot != "" {
				subFmt = fmt.Sprintf("_{ %v }", subSlot)
			}
			if supSlot != "" {
				supFmt = fmt.Sprintf("^{ %v }", supSlot)
			}

			//组成整体公式
			tmplStr := fmt.Sprintf("%v  %v", subFmt, supFmt)
			buf.WriteString(tmplStr)

			//返回数据
			return buf.String(), nil
		case tmVEC:
			/*
			variations：
			variation	symbol	description
			0×0001	tvVE_LEFT	arrow points left
			0×0002	tvVE_RIGHT	arrow points right
			0×0004	tvVE_UNDER	arrow under slot, else over slot
			0×0008	tvVE_HARPOON	harpoon

			这个转换是通过掩码计算的：
			比如variation的值是3，即0000 0000 0000 0011

			对应的是0×0001和0×0002：
			0000 0000 0000 0001
			0000 0000 0000 0010
			*/

			//读取数据 HatBoxClass
			mainAST := ast.children[0]

			//读取latex数据
			mainSlot, _ := m.makeLatex(mainAST)

			//转成latex代码
			var mainStr string
			if mainSlot != "" {
				mainStr = fmt.Sprintf("{ %v }", mainSlot)
			}

			/*
			variation转码
			 */
			variationsMap := make(map[uint16]string)
			variationsMap[0x0001] = "left"
			variationsMap[0x0002] = "right"
			variationsMap[0x0004] = "tvVE_UNDER"
			variationsMap[0x0008] = "harpoonup"

			//有序循环
			variationsCode := []uint16{0x0001, 0x0002, 0x0004, 0x0008}

			topStr := "\\overset\\"
			for _, vCode := range variationsCode {
				if vCode&uint16(tmpl.variation) != 0 {
					topStr = topStr + variationsMap[vCode]
				}
			}

			//如果variationCode小于8，则一定不是harpoon,那么默认就使用arrow
			if tmpl.variation < 8 {
				topStr = topStr + "arrow"
			}
			/*
			variation转码 END
			 */

			//组成整体公式
			tmplStr := fmt.Sprintf("%v %v", topStr, mainStr)
			buf.WriteString(tmplStr)

			return buf.String(), nil
		case tmHAT:
			//读取数据 HatBoxClass
			mainAST := ast.children[0]
			topAST := ast.children[1]

			//读取latex数据
			mainSlot, _ := m.makeLatex(mainAST)
			topSlot, _ := m.makeLatex(topAST)

			//转成latex代码
			var mainStr, topStr string
			if mainSlot != "" {
				mainStr = fmt.Sprintf("{ %v }", mainSlot)
			}
			if topSlot != "" {
				topStr = fmt.Sprintf(" %v ", topSlot)
			}

			//组成整体公式
			tmplStr := fmt.Sprintf("%v %v", topStr, mainStr)
			buf.WriteString(tmplStr)

			return buf.String(), nil
		case tmARC:
			//读取数据 HatBoxClass
			mainAST := ast.children[0]
			topAST := ast.children[1]

			//读取latex数据
			mainSlot, _ := m.makeLatex(mainAST)
			topSlot, _ := m.makeLatex(topAST)

			//转成latex代码
			var mainStr, topStr string
			if mainSlot != "" {
				mainStr = fmt.Sprintf("{ %v }", mainSlot)
			}
			if topSlot != "" {
				topStr = fmt.Sprintf("\\overset %v", topSlot)
			}

			//组成整体公式
			tmplStr := fmt.Sprintf("%v %v", topStr, mainStr)
			buf.WriteString(tmplStr)

			return buf.String(), nil
		default:
			log.Println("TMPL NOT IMPLEMENT", tmpl.selector, tmpl.variation)
		}
		for _, _ast := range ast.children {
			_latex, _ := m.makeLatex(_ast)
			buf.WriteString(_latex)
		}
		return buf.String(), nil
	case PILE:
		for idx, _ast := range ast.children {
			_latex, _ := m.makeLatex(_ast)

			//多个line字符串数据以 \\ 分割
			if idx > 0 {
				buf.WriteString(" \\\\ ")
			}

			buf.WriteString(_latex)
		}
		return buf.String(), nil
	case MATRIX:
		matrixCol := int(ast.value.(*MtMatrix).cols)
		for idx, _ast := range ast.children {
			_latex, _ := m.makeLatex(_ast)

			if idx == 0 {
				buf.WriteString(" \\begin{array} {} ")
				continue
			}

			buf.WriteString(_latex)

			if idx%matrixCol == 0 {
				buf.WriteString(" \\\\ ")
			} else {
				buf.WriteString(" & ")
			}
		}

		buf.WriteString(" \\end{array} ")
		return buf.String(), nil
	case LINE:
		for _, _ast := range ast.children {
			_latex, _ := m.makeLatex(_ast)
			buf.WriteString(_latex)
		}
		return buf.String(), nil
	case EMBELL:
		embellType := EmbellType(ast.value.(*MtEmbellRd).embellType)
		var embellStr string

		switch embellType {
		case emb1DOT:
			embellStr = " \\dot "
		case emb1PRIME:
			embellStr = "'"
		case emb2PRIME:
			embellStr = "''"
		case emb3PRIME:
			embellStr = "'''"
		case embHAT:
			embellStr = " \\hat "
		case embOBAR:
			embellStr = " \\bar "
		default:
			log.Println("not implement embell:", embellType)
		}

		buf.WriteString(embellStr)
		return buf.String(), nil
	}

	return "", nil
}

//[MTEF Storage](https://docs.wiris.com/en/mathtype/mathtype_desktop/mathtype-sdk/mtefstorage)
func Open(reader io.ReadSeeker) (eqn *MTEFv5, err error) {
	//parse `mtef` stream from `ole` object
	ole, err := ole2.Open(reader, "")
	if err != nil {
		fmt.Println(err)
	}

	dir, err := ole.ListDir()
	if err != nil {
		fmt.Println(err)
	}

	for _, file := range dir {
		if "Equation Native" == file.Name() {
			root := dir[0]
			reader := ole.OpenFile(file, root)

			hdrBuffer := make([]byte, oleCbHdr)
			if _, err := reader.Read(hdrBuffer); err == nil {
				hdrReader := bytes.NewReader(hdrBuffer)
				var cbHdr = uint16(0)
				var cbSize = uint32(0)

				_ = binary.Read(hdrReader, binary.LittleEndian, &cbHdr)
				if cbHdr != oleCbHdr {
					return nil, err
				}

				//ignore `version: u32` and `cf: u16`
				_, _ = hdrReader.Seek(4+2, io.SeekCurrent)
				_ = binary.Read(hdrReader, binary.LittleEndian, &cbSize)

				//body from `cbHdr` to `cbHdr + cbSize`
				eqnBody := make([]byte, cbSize);
				_, _ = reader.Seek(int64(cbHdr), io.SeekStart)
				_, _ = reader.Read(eqnBody)

				eqn = new(MTEFv5)
				eqn.reader = bytes.NewReader(eqnBody)
				_ = eqn.readRecord()
				_ = eqn.makeAST()
				return eqn, nil
			}

			return nil, err
		}
	}
	return nil, err
}
