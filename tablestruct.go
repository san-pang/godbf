package godbf

/*
	DBF文件结构说明：
	文件头 32位长度
	字段属性说明	每个字段32位长度
	文件头结束符	1位长度
	数据记录		每一行记录，第1位是删除标志位，其余为实际的数据内容
	文件结束符  1位长度
*/

type fileType uint8
const (
	foxBASE fileType = 0x02               //0x02    FoxBASE
	foxBASE_III_NoMemo fileType = 0x03    //0x03    FoxBASE+/Dbase III plus, no memo
	foxPro fileType = 0x30                //0x30    Visual FoxPro
	foxProAutoincrement fileType = 0x31   //0x31    Visual FoxPro,  enabled autoincrement
	dbase_IV_SQL_Table_NoMemo fileType = 0x43      //0x43    dBASE IV SQL table files, no memo
	dbase_IV_SQL_System_NoMemo fileType = 0x63     //0x63    dBASE IV SQL system files, no memo
	foxBASE_III_Memo fileType = 0x83               //0x83    FoxBASE+/dBASE III PLUS, with memo
	dbase_IV_Memo fileType = 0x8B                  //0x8B    dBASE IV with memo
	dbase_IV_SQL_Table_Memo fileType = 0xCB        //0xCB    dBASE IV SQL table files, with memo
	foxPro2_Memo fileType = 0xF5                   //0xF5    FoxPro 2.x (or earlier) with memo
	foxBASE2 fileType = 0xFB                       //0xFB    FoxBASE
)

type fieldType byte
const (
	fieldtype_character fieldType = 'C'
	fieldtype_logical   fieldType = 'L'  // 布尔BOOL
	fieldtype_date      fieldType = 'D'
	fieldtype_numeric   fieldType = 'N'  // 数值，包括整数和浮点小数
	fieldtype_float     fieldType = 'F'
	/*
		暂不支持
		fieldtype_dateTime  fieldType = "T"
		fieldtype_currency  fieldType = "Y"
		fieldtype_double    fieldType = "B"
		fieldtype_integer   fieldType = "I"
		fieldtype_memo   	fieldType = "M"
		fieldtype_general   fieldType = "G"
		fieldtype_picture   fieldType = "P"
	*/
)

const headerTerminator byte = 0x0D  //文件头的结束符号
const deletedFlag byte = 0x2A  //记录删除标记
const space byte = 0x20  //空格
const null byte = 0x00  //null
const fileTerminator = 0x1A  //文件的结束符号

type dbfHeader struct {
	fileType uint8  //第1位，文件类型
	updateYear uint8  //第2位， 文件修改日期的年，格式YY
	updateMonth uint8 //第3位，文件修改日期的月
	updateDay uint8 //第4位，文件修改日期的日
	recordCount uint32  //第5-8位，数据记录条数
	dataOffset uint16  //第9-10位，数据记录开始的位置
	recordSize uint16 //第11-12位，每一条数据的长度（包括删除标记）
	reserved []byte  //第12-32位，Reserved数据
}

type dbfField struct {
	name          string  //第1-11位，字段名，最大10位，如果不足10位，用0x00填充
	fieldType     fieldType  //第12位，字段类型
	displacement  uint32  //第13-16位，Displacement of field in record
	length        uint8  //第17位，字段长度
	decimalPlaces uint8  //第18位，字段小数位数
	flag		  byte   //第19位，字段标志
	autoincrementNext uint32  //第20-23位，自增字段的下一个值
	autoincrementStep uint8  //第24位，自增步长
	reserved    []byte  //第25-32位，reserve数据
}
