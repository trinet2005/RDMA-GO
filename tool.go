package RDMAGO

import "C"
import "os"

func GetQPInfo(ibRes *IBRes) (*QPInfo, error) {
	return &QPInfo{
		QpNum: ibRes.Qp.qp_num,
		Lid:   ibRes.PortAttr.lid,
		Gid:   ibRes.Gid,
	}, nil
}

func GetFileMeta(fileName string, chunkSize int64) (int, int64, *os.File, error) {
	// 打开文件
	file, err := os.Open(fileName)
	if err != nil {
		return 0, 0, nil, err
	}

	// 获取文件大小
	fileInfo, err := file.Stat()
	if err != nil {
		file.Close()
		return 0, 0, nil, err
	}
	fileSize := fileInfo.Size()

	// 计算分片数
	chunkCount := int(fileSize / chunkSize)
	if fileSize%chunkSize != 0 {
		chunkCount++ // 如果文件大小不是分片大小的整数倍，增加一分片
	}
	return chunkCount, fileSize, file, nil
}
