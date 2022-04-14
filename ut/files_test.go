package ut

import "testing"

func TestSaveSliceToFiles(t *testing.T) {
	SaveSliceToFiles("./test.txt", []interface{}{"测试第一行", "测试第二行"}, "表头")
}
