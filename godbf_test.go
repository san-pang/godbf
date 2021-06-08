package godbf

import (
	"strconv"
	"testing"
)

func BenchmarkNewDBF_Append(b *testing.B) {
	dbf := NewFile("./testdata/test_append.dbf", "gbk")
	defer dbf.Close()
	dbf.AddDateField("BEGIN_DATE")
	dbf.AddDateField("END_DATE")
	dbf.AddFloatField("PRICE", 12, 2)
	dbf.AddNumericField("QTY", 8, 2)
	dbf.AddBooleanField("FINISHED")
	dbf.AddStringField("STOCK_CODE", 20)
	for i:=0; i<b.N; i++ {
		dbf.Append()
		dbf.SetFieldValue("BEGIN_DATE", "20201213")
		dbf.SetFieldValue("END_DATE", "20210605")
		dbf.SetFieldValue("PRICE", "12.34")
		dbf.SetFieldValue("QTY", strconv.FormatInt(int64(i), 10))
		dbf.SetFieldValue("STOCK_CODE", "600570")
		dbf.SetFieldValue("FINISHED", "1")
		dbf.Post()
	}
}

func BenchmarkDBF_Next(b *testing.B) {
	dbf, err := LoadFrom("./testdata/test_5million.DBF", "gbk")
	if err != nil {
		panic(err)
	}
	defer dbf.Close()
	for i := 0; i < b.N; i ++ {
		dbf.Next()
		_ = dbf.StringValueByNameX("BEGIN_DATE")
		_ = dbf.StringValueByNameX("END_DATE")
		_ = dbf.StringValueByNameX("PRICE")
		_ = dbf.StringValueByNameX("QTY")
		_ = dbf.StringValueByNameX("FINISHED")
		_ = dbf.StringValueByNameX("STOCK_CODE")
	}
}

func BenchmarkDBF_Update(b *testing.B) {
	dbf, err := LoadFrom("./testdata/test_5million.DBF", "gbk")
	if err != nil {
		panic(err)
	}
	defer dbf.Close()
	for i := 0; i < b.N; i ++ {
		dbf.Next()
		dbf.SetFieldValue("BEGIN_DATE", "20060123")
		dbf.SetFieldValue("END_DATE", "20060124")
		dbf.SetFieldValue("PRICE", "22.22")
		dbf.SetFieldValue("PRICE", strconv.FormatInt(int64(i+10), 10))
		dbf.SetFieldValue("FINISHED", "0")
		dbf.SetFieldValue("STOCK_CODE", "000002")
		dbf.Post()
	}
}