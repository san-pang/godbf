# godbf
pure Go library with filelock for reading, writing and creating dBase/xBase database files

filelock provides a cross-process mutex to ensure the data safety, reference to [go-filelock](https://github.com/zbiljic/go-filelock)

__ATTENTION PLEASE: only support reading/writing single record once__


# Installation
```
go get github.com/san-pang/godbf
```

# Example:  
## traversal all records, read or update single record
```
import github.com/san-pang/godbf

dbf, err := LoadFrom("./testdata/ZRTBDQXFL.DBF", "gbk")
if err != nil {
	panic(err)
}
defer dbf.Close()
for !dbf.EOF() {
	if err := dbf.Next(); err != nil {
		panic(err)
	}
	// read record
	_ = dbf.StringValueByNameX("jllx")
	_ = dbf.StringValueByNameX("scdm")
	_ = dbf.StringValueByNameX("zqdm")
	_ = dbf.StringValueByNameX("qx")
	_ = dbf.StringValueByNameX("rrfl")
	_ = dbf.StringValueByNameX("rcfl")
	_ = dbf.StringValueByNameX("jyrq")
	// update record, use Post() method to post changes to file
	if err := dbf.SetFieldValue("jllx", "2"); err != nil {
		panic(err)
	}
	if err := dbf.SetFieldValue("rrfl", "0.0130000"); err != nil {
		panic(err)
	}
	if err := dbf.Post(); err != nil {
		panic(err)
	}
}
```

## go to specific record
```
import github.com/san-pang/godbf

dbf, err := LoadFrom("./testdata/ZRTBDQXFL.DBF", "gbk")
if err != nil {
	panic(err)
}
defer dbf.Close()
// record number should be 1 to total counts
if err := dbf.Go(3); err != nil {
	panic(err)
}
// read record
_ = dbf.StringValueByNameX("jllx")
_ = dbf.StringValueByNameX("scdm")
_ = dbf.StringValueByNameX("zqdm")
_ = dbf.StringValueByNameX("qx")
_ = dbf.StringValueByNameX("rrfl")
_ = dbf.StringValueByNameX("rcfl")
_ = dbf.StringValueByNameX("jyrq")
// update record, use Post() method to post changes to file
if err := dbf.SetFieldValue("jllx", "2"); err != nil {
	panic(err)
}
if err := dbf.SetFieldValue("rrfl", "0.0130000"); err != nil {
	panic(err)
}
if err := dbf.Post(); err != nil {
	panic(err)
}
```

## append record to the end
```
import github.com/san-pang/godbf

dbf, err := LoadFrom("./testdata/ZRTBDQXFL.DBF", "gbk")
if err != nil {
	panic(err)
}
defer dbf.Close()
// append record, use Post() method to post changes to file
dbf.Append()
if err := dbf.SetFieldValue("jllx", "2"); err != nil {
	panic(err)
}
if err := dbf.SetFieldValue("rrfl", "0.0130000"); err != nil {
	panic(err)
}
if err := dbf.Post(); err != nil {
	panic(err)
}
```

## create new file with no record
```
import github.com/san-pang/godbf

dbf := NewFile("./testdata/test_newfile.DBF", "gbk")
defer dbf.Close()
dbf.AddDateField("BEGIN_DATE")
dbf.AddDateField("END_DATE")
dbf.AddFloatField("PRICE", 12, 2)
dbf.AddNumericField("QTY", 8, 2)
dbf.AddBooleanField("FINISHED")
dbf.AddStringField("STOCK_CODE", 20)
if err = dbf.SaveNewFile(); err != nil {
	return err
}
```

## create new file with new record
```
import github.com/san-pang/godbf

dbf := NewFile("./testdata/test_newfile.DBF", "gbk")
defer dbf.Close()
dbf.AddDateField("BEGIN_DATE")
dbf.AddDateField("END_DATE")
dbf.AddFloatField("PRICE", 12, 2)
dbf.AddNumericField("QTY", 8, 2)
dbf.AddBooleanField("FINISHED")
dbf.AddStringField("STOCK_CODE", 20)
// append record, use Post() method to post changes to file
dbf.Append()
if err := dbf.SetFieldValue("QTY", "2"); err != nil {
	panic(err)
}
if err := dbf.SetFieldValue("PRICE", "12.13"); err != nil {
	panic(err)
}
if err := dbf.Post(); err != nil {
	panic(err)
}
```

# benchmark
```
goos: windows
goarch: amd64
cpu: Intel(R) Core(TM) i7-8700 CPU @ 3.20GHz
BenchmarkNewDBF_Append-12          69056             17114 ns/op
BenchmarkDBF_Next-12              263623              4512 ns/op
BenchmarkDBF_Update-12            106771             11013 ns/op
```
