package godbf

import (
	"bytes"
	"encoding/binary"
	"github.com/axgle/mahonia"
	"github.com/shopspring/decimal"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type DBF struct {
	head dbfHeader
	headBuff []byte
	file *os.File
	filename string
	currentRecordNo uint32
	fieldsMap map[string]dbfField
	fieldsList []dbfField
	fieldsCount int
	recordBuff []byte
	eof bool
	encoder mahonia.Encoder
	decoder mahonia.Decoder
	append bool
	filelock tryLockerSafe
}

func LoadFrom(filename string, encoding string) (dbf *DBF, err error) {
	f, err := os.OpenFile(filename, os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	dbf = &DBF{
		headBuff: make([]byte, 32),
		file:     f,
		filename: filename,
		currentRecordNo: 0,
		eof: true,
		encoder: mahonia.NewEncoder(encoding),
		decoder: mahonia.NewDecoder(encoding),
		append: false,
		filelock: newLock(f),
	}
	err = dbf.readHead()
	if err != nil {
		return nil, err
	}
	err = dbf.readFields()
	if err != nil {
		return nil, err
	}
	dbf.recordBuff = bytes.Repeat([]byte{space}, int(dbf.head.recordSize))
	dbf.eof = dbf.head.recordCount == 0
	return dbf, nil
}

func (dbf *DBF)readHead() error {
	_, err := dbf.file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	_, err = dbf.file.Read(dbf.headBuff)
	if err != nil {
		return err
	}
	dbf.head = dbfHeader{
		fileType:    dbf.headBuff[0],
		updateYear:  dbf.headBuff[1],
		updateMonth: dbf.headBuff[2],
		updateDay:   dbf.headBuff[3],
		recordCount: binary.LittleEndian.Uint32(dbf.headBuff[4:8]),
		dataOffset:  binary.LittleEndian.Uint16(dbf.headBuff[8:10]),
		recordSize:  binary.LittleEndian.Uint16(dbf.headBuff[10:12]),
		reserved:    dbf.headBuff[12:32],
	}
	return nil
}

func (dbf *DBF)readFields() error {
	//先偏移到字段结构开始的地方
	_, err := dbf.file.Seek(32, io.SeekStart)
	if err != nil {
		return err
	}
	// 字段个数，每个字段32位，DBF文件头固定32位，文件头结束标志0x0D占1位
	fieldsCount := (dbf.head.dataOffset - 32 -1) / 32
	dbf.fieldsMap = make(map[string]dbfField, fieldsCount)
	for i:=0; i<int(fieldsCount); i++ {
		fieldBuff := make([]byte, 32)
		_, err = dbf.file.Read(fieldBuff)
		if err != nil {
			return err
		}
		field := dbfField{
			name:              dbf.decoder.ConvertString(strings.TrimSpace(strings.Trim(bytes2str(fieldBuff[:11]), bytes2str([]byte{0})))),
			fieldType:         fieldType(fieldBuff[11]),
			displacement:      binary.LittleEndian.Uint32(fieldBuff[12:16]),
			length:            fieldBuff[16],
			decimalPlaces:     fieldBuff[17],
			flag:              fieldBuff[18],
			autoincrementNext: binary.LittleEndian.Uint32(fieldBuff[19:23]),
			autoincrementStep: fieldBuff[23],
			reserved:          fieldBuff[24:32],
		}
		dbf.fieldsList = append(dbf.fieldsList, field)
		dbf.fieldsMap[field.name] = field
	}
	dbf.fieldsCount = len(dbf.fieldsList)
	return nil
}

func (dbf *DBF)Go(recordNo uint32) error {
	// 定位的行数，从1开始，以数据条数结尾
	if recordNo <= 0 {
		return record_index_out_of_range
	}
	//如果发现到文件尾了，重新读取一下文件头，有可能有新数据写进来
	if recordNo > dbf.head.recordCount {
		if err := dbf.readHead(); err != nil {
			return err
		}
	}
	//重新读取之后还是空的，那就报错
	if recordNo > dbf.head.recordCount {
		return record_index_out_of_range
	}
	//读取数据记录
	_, err := dbf.file.Seek(int64(dbf.head.dataOffset) + int64(recordNo - 1) * int64(dbf.head.recordSize), io.SeekStart)
	if err != nil {
		return err
	}
	_, err = dbf.file.Read(dbf.recordBuff)
	if err != nil {
		return err
	}
	dbf.currentRecordNo = recordNo
	dbf.eof = dbf.currentRecordNo >= dbf.head.recordCount
	return nil
}

func (dbf *DBF)EOF() bool {
	return dbf.eof
}

func (dbf *DBF)First() error {
	return dbf.Go(1)
}

func (dbf *DBF)Last() error {
	return dbf.Go(dbf.head.recordCount)
}

func (dbf *DBF)Next() error {
	return dbf.Go(dbf.currentRecordNo + 1)
}

func (dbf *DBF)Close() error {
	if dbf.file != nil {
		return dbf.file.Close()
	}
	return nil
}

func (dbf *DBF)RecordCount() uint32 {
	return dbf.head.recordCount
}

func (dbf *DBF)FieldNames() []string {
	var fieldNames []string
	for _, f := range dbf.fieldsList {
		fieldNames = append(fieldNames, f.name)
	}
	return fieldNames
}

func (dbf *DBF)FieldsCount() int {
	return dbf.fieldsCount
}

func (dbf *DBF)StringValueByName(fieldname string) (value string, err error) {
	field, ok := dbf.fieldsMap[fieldname]
	if !ok {
		return "", field_not_exists
	}
	return strings.TrimSpace(dbf.decoder.ConvertString(bytes2str(dbf.recordBuff[field.displacement: field.displacement+uint32(field.length)]))), nil
}

func (dbf *DBF)DecimalValueByName(fieldname string) (value decimal.Decimal, err error) {
	field, ok := dbf.fieldsMap[fieldname]
	if !ok {
		return decimal.Zero, field_not_exists
	}
	return decimal.NewFromString(strings.TrimSpace(dbf.decoder.ConvertString(bytes2str(dbf.recordBuff[field.displacement: field.displacement+uint32(field.length)]))))
}

func (dbf *DBF)DecimalValueByNameX(fieldname string) (value decimal.Decimal) {
	field, ok := dbf.fieldsMap[fieldname]
	if !ok {
		return decimal.Zero
	}
	value, _ = decimal.NewFromString(strings.TrimSpace(dbf.decoder.ConvertString(bytes2str(dbf.recordBuff[field.displacement: field.displacement+uint32(field.length)]))))
	return value
}

func (dbf *DBF)StringValueByNameX(fieldname string) (value string) {
	field, ok := dbf.fieldsMap[fieldname]
	if !ok {
		return ""
	}
	return strings.TrimSpace(dbf.decoder.ConvertString(bytes2str(dbf.recordBuff[field.displacement: field.displacement+uint32(field.length)])))
}

func (dbf *DBF)IntValueByName(fieldname string) (value int, err error) {
	field, ok := dbf.fieldsMap[fieldname]
	if !ok {
		return 0, field_not_exists
	}
	return strconv.Atoi(strings.TrimSpace(dbf.decoder.ConvertString(bytes2str(dbf.recordBuff[field.displacement: field.displacement+uint32(field.length)]))))
}

func (dbf *DBF)IntValueByNameX(fieldname string) (value int) {
	field, ok := dbf.fieldsMap[fieldname]
	if !ok {
		return 0
	}
	value, _ = strconv.Atoi(strings.TrimSpace(dbf.decoder.ConvertString(bytes2str(dbf.recordBuff[field.displacement: field.displacement+uint32(field.length)]))))
	return value
}

func (dbf *DBF)FloatValueByName(fieldname string) (value float64, err error) {
	field, ok := dbf.fieldsMap[fieldname]
	if !ok {
		return 0, field_not_exists
	}
	return strconv.ParseFloat(strings.TrimSpace(dbf.decoder.ConvertString(bytes2str(dbf.recordBuff[field.displacement: field.displacement+uint32(field.length)]))), 64)
}

func (dbf *DBF)FloatValueByNameX(fieldname string) (value float64) {
	field, ok := dbf.fieldsMap[fieldname]
	if !ok {
		return 0
	}
	value, _ = strconv.ParseFloat(strings.TrimSpace(dbf.decoder.ConvertString(bytes2str(dbf.recordBuff[field.displacement: field.displacement+uint32(field.length)]))), 64)
	return value
}

func (dbf *DBF)IsDeleted() bool {
	return dbf.recordBuff[0] == deletedFlag
}

func (dbf *DBF)Append()  {
	dbf.append = true
	dbf.recordBuff = bytes.Repeat([]byte{space}, int(dbf.head.recordSize))
	for _, field := range dbf.fieldsList {
		switch field.fieldType {
		case fieldtype_float:
			copy(dbf.recordBuff[field.displacement: field.displacement+uint32(field.length)], strconv.FormatFloat(0, 'f', int(field.decimalPlaces), 64))
		case fieldtype_logical:
			copy(dbf.recordBuff[field.displacement: field.displacement+uint32(field.length)], strconv.FormatFloat(0, 'f', int(field.decimalPlaces), 64))
		case fieldtype_numeric:
			copy(dbf.recordBuff[field.displacement: field.displacement+uint32(field.length)], strconv.FormatFloat(0, 'f', int(field.decimalPlaces), 64))
		default:
			//其余的全部当成字符串处理, 不需要做任何操作，默认空字符串
		}
	}
}

func (dbf *DBF)SetFieldValue(fieldname string, value string) error {
	field, ok := dbf.fieldsMap[fieldname]
	if !ok {
		return field_not_exists
	}
	copy(dbf.recordBuff[field.displacement: field.displacement + uint32(field.length)], str2bytes(dbf.encoder.ConvertString(value)))
	return nil
}

func (dbf *DBF)Post() (err error) {
	if dbf.fieldsCount <= 0 {
		return empty_fields
	}
	// 有可能是新增文件之后没有保存，就直接新增数据提交，这种情况下需要先保存文件
	if dbf.file == nil {
		if err = dbf.SaveNewFile(); err != nil {
			return err
		}
	}
	if err = dbf.filelock.lock(); err != nil {
		return err
	}
	defer dbf.filelock.unlock()
	if !dbf.append {
		// update
		_, err = dbf.file.WriteAt(dbf.recordBuff, int64(dbf.head.dataOffset) + int64(dbf.currentRecordNo - 1) * int64(dbf.head.recordSize))
		return nil
	}
	// 新增数据
	// 先把数据写进去，需要重新读取一下头部，不然有可能加锁写之前，有其它进程已经写了新数据进来
	if err = dbf.readHead(); err != nil {
		return err
	}
	if _, err = dbf.file.WriteAt(append(dbf.recordBuff, fileTerminator), int64(dbf.head.dataOffset) + int64(dbf.head.recordCount) * int64(dbf.head.recordSize)); err != nil {
		return err
	}
	//更新头信息里面的数据条数
	recordCountBuff := make([]byte, 4)
	binary.LittleEndian.PutUint32(recordCountBuff, dbf.head.recordCount+1)
	_, err = dbf.file.WriteAt(recordCountBuff, 4)
	return err
}

func NewFile(filename string, encoding string) *DBF {
	return &DBF{
		head:           dbfHeader{
			fileType:    byte(foxBASE_III_NoMemo),
			updateYear:  byte(time.Now().Year() - 1900),
			updateMonth: byte(time.Now().Month()),
			updateDay:   byte(time.Now().Day()),
			recordCount: 0,
			dataOffset:  0,
			recordSize:  1,  //删除标记
			reserved:    nil,
		},
		headBuff:        make([]byte, 32),
		file:            nil,
		filename:        filename,
		currentRecordNo: 0,
		fieldsMap:       make(map[string]dbfField),
		fieldsList:      make([]dbfField, 0),
		fieldsCount:     0,
		recordBuff:      make([]byte, 0),
		eof:             true,
		encoder:         mahonia.NewEncoder(encoding),
		decoder:         mahonia.NewDecoder(encoding),
		append:          false,
		filelock:        nil,
	}
}

func (dbf *DBF) addField(fieldName string, fieldType fieldType, length uint8, precision uint8) {
	// 第1位是删除标记
	var displacement uint32 = 1
	if dbf.fieldsCount > 0 {
		prevField := dbf.fieldsList[dbf.fieldsCount - 1]
		displacement = prevField.displacement + uint32(prevField.length)
	}
	field := dbfField{
		name:              fieldName,
		fieldType:         fieldType,
		displacement:      displacement,
		length:            length,
		decimalPlaces:     precision,
		flag:              0,
		autoincrementNext: 0,
		autoincrementStep: 0,
		reserved:          make([]byte, 20),
	}
	dbf.fieldsList = append(dbf.fieldsList, field)
	dbf.fieldsCount += 1
	dbf.fieldsMap[field.name] = field
	dbf.head.recordSize = dbf.head.recordSize + uint16(length)
	//32位长度的header + 字段描述个数 * 每个字段描述32位长度 + 1位文件头结束符
	dbf.head.dataOffset = uint16(32 + 32 * dbf.fieldsCount + 1)
}

func (dbf *DBF)AddBooleanField(fieldName string) {
	dbf.addField(fieldName, fieldtype_logical, 1, 0)
}

func (dbf *DBF)AddDateField(fieldName string) {
	dbf.addField(fieldName, fieldtype_date, 8, 0)
}

func (dbf *DBF)AddStringField(fieldName string, length uint8) {
	dbf.addField(fieldName, fieldtype_character, length, 0)
}

func (dbf *DBF)AddNumericField(fieldName string, length uint8, precision uint8) {
	dbf.addField(fieldName, fieldtype_numeric, length, precision)
}

func (dbf *DBF)AddFloatField(fieldName string, length uint8, precision uint8) {
	dbf.addField(fieldName, fieldtype_float, length, precision)
}

func (dbf *DBF)FileName() string {
	return dbf.filename
}

func (dbf *DBF)SaveNewFile() (err error) {
	// 32位文件头，dbf.fieldsCount * 32位字段长度，1位文件头结束标记，1位文件尾结束标记
	fileBuff := make([]byte, 32 + dbf.fieldsCount * 32 + 1 + 1)
	// 新文件的头
	fileBuff[0] = dbf.head.fileType
	fileBuff[1] = dbf.head.updateYear
	fileBuff[2] = dbf.head.updateMonth
	fileBuff[3] = dbf.head.updateDay
	binary.LittleEndian.PutUint32(fileBuff[4:8], dbf.head.recordCount)
	binary.LittleEndian.PutUint16(fileBuff[8:10], dbf.head.dataOffset)
	binary.LittleEndian.PutUint16(fileBuff[10:12], dbf.head.recordSize)
	copy(fileBuff[12:32], dbf.head.reserved)
	// 字段描述
	// 字段名，最大10位，如果不足10位，用0x00填充
	blankFieldName := bytes.Repeat([]byte{null}, 10)
	for i:=0; i<dbf.fieldsCount;i++ {
		field := dbf.fieldsList[i]
		copy(fileBuff[32 + i*32: 32 + i*32 + 11], blankFieldName)
		copy(fileBuff[32 + i*32: 32 + i*32 + 11], str2bytes(field.name))
		fileBuff[32 + i*32 + 11] = byte(field.fieldType)
		binary.LittleEndian.PutUint32(fileBuff[32 + i*32 + 12: 32 + i*32 + 16], field.displacement)
		fileBuff[32 + i*32 + 16] = field.length
		fileBuff[32 + i*32 + 17] = field.decimalPlaces
		fileBuff[32 + i*32 + 18] = field.flag
		binary.LittleEndian.PutUint32(fileBuff[32 + i*32 + 19: 32 + i*32 + 23], field.autoincrementNext)
		fileBuff[32 + i*32 + 23] = field.autoincrementStep
		copy(fileBuff[32 + i*32 + 24: 32 + i*32 + 32], blankFieldName)
	}
	fileBuff[len(fileBuff)-2] = headerTerminator
	fileBuff[len(fileBuff)-1] = fileTerminator
	dbf.headBuff = fileBuff[:32]
	if len(dbf.recordBuff) == 0 {
		dbf.recordBuff = bytes.Repeat([]byte{space}, int(dbf.head.recordSize))
	}
	f, err := os.OpenFile(dbf.filename, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	dbf.file = f
	dbf.filelock = newLock(dbf.file)
	if err = dbf.filelock.lock(); err != nil {
		return err
	}
	defer dbf.filelock.unlock()
	_, err = dbf.file.Write(fileBuff)
	return err
}