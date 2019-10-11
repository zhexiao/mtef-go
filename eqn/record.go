package eqn

import "fmt"

type RecordType uint8
type OptionType uint8

const (
	END            RecordType = 0
	LINE           RecordType = 1
	CHAR           RecordType = 2
	TMPL           RecordType = 3
	PILE           RecordType = 4
	MATRIX         RecordType = 5
	EMBELL         RecordType = 6
	RULER          RecordType = 7
	FONT_STYLE_DEF RecordType = 8
	SIZE           RecordType = 9
	FULL           RecordType = 10
	SUB            RecordType = 11
	SUB2           RecordType = 12
	SYM            RecordType = 13
	SUBSYM         RecordType = 14
	COLOR          RecordType = 15
	COLOR_DEF      RecordType = 16
	FONT_DEF       RecordType = 17
	EQN_PREFS      RecordType = 18
	ENCODING_DEF   RecordType = 19
	FUTURE         RecordType = 100
	ROOT           RecordType = 255
)

const (
	MtefOptNudge           OptionType = 0x08
	MtefOptCharEmbell      OptionType = 0x01
	MtefOptCharFuncStart   OptionType = 0x02
	MtefOptCharEncChar8    OptionType = 0x04
	MtefOptCharEncChar16   OptionType = 0x10
	MtefOptCharEncNoMtcode OptionType = 0x20
	MtefOptLineNull        OptionType = 0x01
	mtefOPT_LP_RULER		OptionType = 0x02
	MtefOptLineLspace      OptionType = 0x04
	MtefOptLpRuler         OptionType = 0x02
	MtefColorCmyk          OptionType = 0x01
	MtefColorSpot          OptionType = 0x02
	MtefColorName          OptionType = 0x04
	mtefCOLOR_CMYK         OptionType = 0x01
	mtefCOLOR_SPOT         OptionType = 0x02
	mtefCOLOR_NAME         OptionType = 0x04
)

const (
	fnTEXT     uint8 = 1
	fnFUNCTION uint8 = 2
	fnVARIABLE uint8 = 3
	fnLCGREEK  uint8 = 4
	fnUCGREEK  uint8 = 5
	fnSYMBOL   uint8 = 6
	fnVECTOR   uint8 = 7
	fnNUMBER   uint8 = 8
	fnUSER1    uint8 = 9
	fnUSER2    uint8 = 10
	fnMTEXTRA  uint8 = 11
	fnTEXT_FE  uint8 = 12
	fnEXPAND   uint8 = 22
	fnMARKER   uint8 = 23
	fnSPACE    uint8 = 24
)

type MtTabStop struct {
	next   *MtTabStop
	_type  int16
	offset int16
}

type MtRuler struct {
	nStops      int16
	tabStopList *MtTabStop
}

type MtLine struct {
	nudgeX     int16
	nudgeY     int16
	lineSpace  uint8
	null       bool
	ruler      *MtRuler
	objectList *MtObjList
}

type MtEmbell struct {
	next   *MtEmbell
	nudgeX int16
	nudgeY int16
	embell uint8
}

type MtChar struct {
	nudgeX   int16
	nudgeY   int16
	options  uint8
	typeface uint8
	//16-bit integer MTCode value
	mtcode uint16
	//8-bit font position
	bits8 uint8
	//16-bit integer font position
	bits16         uint16
	embellishments *MtEmbell
}

type MtEqnPrefs struct {
	sizes  []string
	spaces []string
	styles []byte
}

type MtfontStyleDef struct {
	fontDefIndex uint8
	name         string
}

type MtfontDef struct {
	encDefIndex uint8
	name        string
}

type MtColorDefIndex struct {
	index uint8
}

type MtColorDef struct {
	values []uint8
	name   string
}

type MtObjList struct {
	next   *MtObjList
	tag    RecordType
	objPtr []MtObject
}

type MtTmpl struct {
	nudgeX     int16
	nudgeY     int16
	selector   uint8
	variation  uint16
	options    uint8
	objectList *MtObjList
}

type MtPile struct {
	nudgeX int16
	nudgeY int16
	halign uint8
	valign uint8

	//ruler可以不读，不影响后面字节错位，因为这个是一个完整的额外record数据
	ruler *MtRuler

	//objectList可以不读，不影响后面字节错位，因为这个是一个完整的额外record数据
	objectList *MtObjList
}

type MtMatrix struct {
	nudgeX int16
	nudgeY int16
	valign uint8
	h_just uint8
	v_just uint8

	rows uint8
	cols uint8

	//row_parts uint8
	//col_parts uint8

	//objectList可以不读，不影响后面字节错位，因为这个是一个完整的额外record数据
	objectList *MtObjList
}

type MtEmbellRd struct {
	options    uint8
	nudgeX     int16
	nudgeY     int16
	embellType uint8
}

type MtAST struct {
	tag      RecordType
	value    MtObject
	children []*MtAST
}

type MtObject interface{}

func (ast *MtAST) debug(indent int) {
	fmt.Printf("> %#v MtAST %#v\n", indent, ast)
	indent += 1
	for _, ele := range ast.children {
		ele.debug(indent)
	}
}

type SelectorType uint8

//Template selectors and variations:
const (
	//Fences (parentheses, etc.):
	//selector	symbol	description	class
	tmANGLE   SelectorType = 0 //	angle brackets	ParBoxClass
	tmPAREN   SelectorType = 1 //	parentheses	ParBoxClass
	tmBRACE   SelectorType = 2 //	braces (curly brackets)	ParBoxClass
	tmBRACK   SelectorType = 3 //	square brackets	ParBoxClass
	tmBAR     SelectorType = 4 //	vertical bars	ParBoxClass
	tmDBAR    SelectorType = 5 //	double vertical bars	ParBoxClass
	tmFLOOR   SelectorType = 6 //	floor brackets	ParBoxClass
	tmCEILING SelectorType = 7 //	ceiling brackets	ParBoxClass
	tmOBRACK  SelectorType = 8 //	open (white) brackets	ParBoxClass
	//variations	variation bits	symbol	description
	//0×0001	tvFENCE_L	left fence is present
	//0×0002	tvFENCE_R	right fence is present

	//Intervals:
	//selector	symbol	description	class
	tmINTERVAL SelectorType = 9 //	unmatched brackets and parentheses	ParBoxClass
	//variations	variation bits	symbol	description
	//0×0000	tvINTV_LEFT_LP	left fence is left parenthesis
	//0×0001	tvINTV_LEFT_RP	left fence is right parenthesis
	//0×0002	tvINTV_LEFT_LB	left fence is left bracket
	//0×0003	tvINTV_LEFT_RB	left fence is right bracket
	//0×0000	tvINTV_RIGHT_LP	right fence is left parenthesis
	//0×0010	tvINTV_RIGHT_RP	right fence is right parenthesis
	//0×0020	tvINTV_RIGHT_LB	right fence is left bracket
	//0×0030	tvINTV_RIGHT_RB	right fence is right bracket

	//Radicals (square and nth roots):
	//selector	symbol	description	class
	tmROOT SelectorType = 10 //	radical	RootBoxClass
	//variations	variation	symbol	description
	//0	tvROOT_SQ	square root
	//1	tvROOT_NTH	nth root

	//Fractions:
	//selector	symbol	description	class
	tmFRACT SelectorType = 11 //	fractions
	//variations	variation bits	symbol	description
	//0×0001	tvFR_SMALL	subscript-size slots (piece fraction)
	//0×0002	tvFR_SLASH	fraction bar is a slash
	//0×0004	tvFR_BASE	num. and denom. are baseline aligned

	//Over and Underbars:
	//selector	symbol	description	class
	tmUBAR SelectorType = 12 //	underbar	BarBoxClass
	tmOBAR SelectorType = 13 //	overbar	BarBoxClass
	//variations	variation bits	symbol	description
	//0×0001	tvBAR_DOUBLE	bar is doubled, else single

	//Arrows:
	//selector	symbol	description	class
	tmARROW SelectorType = 14 //	arrow	ArroBoxClass
	//variations	variation	symbol	description
	//0×0000	tvAR_SINGLE	single arrow
	//0×0001	tvAR_DOUBLE	double arrow
	//0×0002	tvAR_HARPOON	harpoon
	//0×0004	tvAR_TOP	top slot is present
	//0×0008	tvAR_BOTTOM	bottom slot is present
	//0×0010	tvAR_LEFT	if single, arrow points left
	//0×0020	tvAR_RIGHT	if single, arrow points right
	//0×0010	tvAR_LOS	if double or harpoon, large over small
	//0×0020	tvAR_SOL	if double or harpoon, small over large

	//Integrals (see Limit Variations):
	//selector	symbol	description	class
	tmINTEG SelectorType = 15 //	integral	BigOpBoxClass
	//variations	variation	symbol	description
	//0×0001	tvINT_1	single integral sign
	//0×0002	tvINT_2	double integral sign
	//0×0003	tvINT_3	triple integral sign
	//0×0004	tvINT_LOOP	has loop w/o arrows
	//0×0008	tvINT_CW_LOOP	has clockwise loop
	//0×000C	tvINT_CCW_LOOP	has counter-clockwise loop
	//0×0100	tvINT_EXPAND	integral signs expand

	//Sums, products, coproducts, unions, intersections, etc. (see Limit Variations):
	//selector	symbol	description	class
	tmSUM    SelectorType = 16 //	sum	BigOpBoxClass
	tmPROD   SelectorType = 17 //	product	BigOpBoxClass
	tmCOPROD SelectorType = 18 //	coproduct	BigOpBoxClass
	tmUNION  SelectorType = 19 //	union	BigOpBoxClass
	tmINTER  SelectorType = 20 //	intersection	BigOpBoxClass
	tmINTOP  SelectorType = 21 //	integral-style big operator	BigOpBoxClass
	tmSUMOP  SelectorType = 22 //	summation-style big operator	BigOpBoxClass

	//Limits (see Limit Variations):
	//selector	symbol	description	class
	tmLIM SelectorType = 23 //	limits	LimBoxClass
	//variations	variation	symbol	description
	//0	tvSUBAR	single underbar
	//1	tvDUBAR	double underbar

	//Horizontal braces and brackets:
	//selector	symbol	description	class
	tmHBRACE SelectorType = 24 //	horizontal brace	HFenceBoxClass
	tmHBRACK SelectorType = 25 //	horizontal bracket	HFenceBoxClass
	//variations	variation	symbol	description
	//0×0001	tvHB_TOP	slot is on the top, else on the bottom

	//Long division:
	//selector	symbol	description	class
	tmLDIV SelectorType = 26 //	long division	LDivBoxClass
	//variations	variation	symbol	description
	//0×0001	tvLD_UPPER	upper slot is present

	//Subscripts and superscripts:
	//selector	symbol	description	class
	tmSUB    SelectorType = 27 //	subscript	ScrBoxClass
	tmSUP    SelectorType = 28 //	superscript	ScrBoxClass
	tmSUBSUP SelectorType = 29 //	subscript and superscript	ScrBoxClass
	//variations	variation	symbol	description
	//0×0001	tvSU_PRECEDES	script precedes scripted item,

	//else follows
	//Dirac bra-ket notation:
	//selector	symbol	description	class
	tmDIRAC SelectorType = 30 //	bra-ket notation	DiracBoxClass
	//variations	variation	symbol	description
	//0×0001	tvDI_LEFT	left part is present
	//0×0002	tvDI_RIGHT	right part is present

	//Vectors:
	//selector	symbol	description	class
	tmVEC SelectorType = 31 //	vector	HatBoxClass
	//variations	variation	symbol	description
	//0×0001	tvVE_LEFT	arrow points left
	//0×0002	tvVE_RIGHT	arrow points right
	//0×0004	tvVE_UNDER	arrow under slot, else over slot
	//0×0008	tvVE_HARPOON	harpoon

	//Hats, arcs, tilde, joint status:
	//selector	symbol	description	class
	tmTILDE   SelectorType = 32 //	tilde over characters	HatBoxClass
	tmHAT     SelectorType = 33 //	hat over characters	HatBoxClass
	tmARC     SelectorType = 34 //	arc over characters	HatBoxClass
	tmJSTATUS SelectorType = 35 //	joint status construct	HatBoxClass

	//Overstrikes (cross-outs):
	//selector	symbol	description	class
	tmSTRIKE SelectorType = 36 //	overstrike (cross-out)	StrikeBoxClass
	//variations	variation	symbol	description
	//0×0001	tvST_HORIZ	line is horizontal, else slashes
	//0×0002	tvST_UP	if slashes, slash from lower-left to upper-right is present
	//0×0004	tvST_DOWN	if slashes, slash from upper-left to lower-right is present

	//Boxes:
	//selector	symbol	description	class
	tmBOX SelectorType = 37 //	box	TBoxBoxClass
	//variations	variation	symbol	description
	//0×0001	tvBX_ROUND	corners are round, else square
	//0×0002	tvBX_LEFT	left side is present
	//0×0004	tvBX_RIGHT	right side is present
	//0×0008	tvBX_TOP	top side is present
	//0×0010	tvBX_BOTTOM	bottom side is present
)

type EmbellType uint8

const (
	emb1DOT      EmbellType = 2
	emb2DOT      EmbellType = 3
	emb3DOT      EmbellType = 4
	emb1PRIME    EmbellType = 5
	emb2PRIME    EmbellType = 6
	embBPRIME    EmbellType = 7
	embTILDE     EmbellType = 8
	embHAT       EmbellType = 9
	embNOT       EmbellType = 10
	embRARROW    EmbellType = 11
	embLARROW    EmbellType = 12
	embBARROW    EmbellType = 13
	embR1ARROW   EmbellType = 14
	embL1ARROW   EmbellType = 15
	embMBAR      EmbellType = 16
	embOBAR      EmbellType = 17
	emb3PRIME    EmbellType = 18
	embFROWN     EmbellType = 19
	embSMILE     EmbellType = 20
	embX_BARS    EmbellType = 21
	embUP_BAR    EmbellType = 22
	embDOWN_BAR  EmbellType = 23
	emb4DOT      EmbellType = 24
	embU_1DOT    EmbellType = 25
	embU_2DOT    EmbellType = 26
	embU_3DOT    EmbellType = 27
	embU_4DOT    EmbellType = 28
	embU_BAR     EmbellType = 29
	embU_TILDE   EmbellType = 30
	embU_FROWN   EmbellType = 31
	embU_SMILE   EmbellType = 32
	embU_RARROW  EmbellType = 33
	embU_LARROW  EmbellType = 34
	embU_BARROW  EmbellType = 35
	embU_R1ARROW EmbellType = 36
	embU_L1ARROW EmbellType = 37
)
