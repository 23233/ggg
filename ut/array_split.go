package ut

// SplitArrayByThreshold 根据固定值分配成数组 例如threshold为5000 则把arr按照5000一个数组进行切分
func SplitArrayByThreshold[T any](arr []T, threshold int) [][]T {
	count := (len(arr) + threshold - 1) / threshold

	// 创建一个二维切片用于存储分割后的子数组
	result := make([][]T, count)

	// 分割数组并存储到结果中
	for i := 0; i < count; i++ {
		start := i * threshold
		end := (i + 1) * threshold
		if end > len(arr) {
			end = len(arr)
		}
		result[i] = arr[start:end]
	}

	return result
}

// SplitArrayByFixedSize 根据限制分配数组 例如fixedBulkSize为10 则会把arr分为10分
func SplitArrayByFixedSize[T any](arr []T, fixedBulkSize int) [][]T {
	count := fixedBulkSize
	if len(arr) < count {
		count = len(arr)
	}

	// 创建一个二维切片用于存储分割后的子数组
	result := make([][]T, count)

	// 分割数组并存储到结果中
	for i := 0; i < len(arr); i++ {
		bucket := i % count
		result[bucket] = append(result[bucket], arr[i])
	}

	return result
}
