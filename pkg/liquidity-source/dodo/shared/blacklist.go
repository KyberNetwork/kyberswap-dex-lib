package shared

import (
	"bufio"
	"bytes"

	"github.com/KyberNetwork/logger"
	cmap "github.com/orcaman/concurrent-map"
)

func InitBlackList(blackListPath string) (cmap.ConcurrentMap, error) {
	blackListMap := cmap.New()

	if blackListPath == "" {
		return blackListMap, nil
	}

	byteData, ok := BytesByPath[blackListPath]
	if !ok {
		logger.WithFields(logger.Fields{
			"blacklistFilePath": blackListPath,
		}).Error(ErrInitializeBlacklistFailed.Error())

		return blackListMap, ErrInitializeBlacklistFailed
	}

	file := bytes.NewReader(byteData)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		poolAddress := scanner.Text()
		if poolAddress != "" {
			blackListMap.Set(poolAddress, true)
		}
	}

	return blackListMap, nil
}
