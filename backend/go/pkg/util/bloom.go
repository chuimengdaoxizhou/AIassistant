package util

import (
	"encoding/gob"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"sync"

	"github.com/bits-and-blooms/bitset"
)

// --- 基础的、非扩容的布隆过滤器 ---

// BloomFilter 定义了一个基础的布隆过滤器结构。
type BloomFilter struct {
	M         uint           // 位数组大小
	K         uint           // 哈希函数数量
	Bits      *bitset.BitSet // 位数组
	ItemCount uint           // 已添加的元素数量
	Capacity  uint           // 预估容量
}

// NewBloomFilter 创建一个基础的布隆过滤器。
// capacity: 预估要存储的元素数量。
// errorRate: 期望的误报率 (例如 0.01 表示 1%)。
func NewBloomFilter(capacity uint, errorRate float64) *BloomFilter {
	m := calculateM(capacity, errorRate)
	k := calculateK(capacity, m)
	return &BloomFilter{
		M:         m,
		K:         k,
		Bits:      bitset.New(m),
		Capacity:  capacity,
		ItemCount: 0,
	}
}

// Add 向布隆过滤器中添加一个元素。
func (bf *BloomFilter) Add(data []byte) {
	hashes := bf.hashKernels(data)
	for i := uint(0); i < bf.K; i++ {
		bf.Bits.Set(uint(hashes[i] % uint64(bf.M)))
	}
	bf.ItemCount++
}

// Test 检查一个元素是否可能存在于布隆过滤器中。
func (bf *BloomFilter) Test(data []byte) bool {
	hashes := bf.hashKernels(data)
	for i := uint(0); i < bf.K; i++ {
		if !bf.Bits.Test(uint(hashes[i] % uint64(bf.M))) {
			return false
		}
	}
	return true
}

// isFull 检查过滤器是否已达到其容量。
func (bf *BloomFilter) isFull() bool {
	return bf.ItemCount >= bf.Capacity
}

// hashKernels 生成 k 个不同的哈希值。
func (bf *BloomFilter) hashKernels(data []byte) []uint64 {
	h1 := fnv.New64a()
	h1.Write(data)
	hash1 := h1.Sum64()

	h2 := fnv.New64()
	h2.Write(data)
	hash2 := h2.Sum64()

	hashes := make([]uint64, bf.K)
	for i := uint(0); i < bf.K; i++ {
		hashes[i] = hash1 + uint64(i)*hash2
	}
	return hashes
}

// m = - (n * log(p)) / (log(2)^2)
func calculateM(n uint, p float64) uint {
	return uint(math.Ceil(-(float64(n) * math.Log(p)) / (math.Pow(math.Log(2), 2))))
}

// k = (m / n) * log(2)
func calculateK(n uint, m uint) uint {
	k := uint(math.Ceil((float64(m) / float64(n)) * math.Log(2)))
	if k < 1 {
		return 1
	}
	return k
}

// --- 可伸缩布隆过滤器 (SBF) ---

// SBFConfig 定义了SBF的配置参数。
// 字段已设为可导出，以便 gob 序列化。
type SBFConfig struct {
	InitialCapacity      uint
	ErrorRate            float64
	GrowthFactor         float64
	ErrorTighteningRatio float64
}

// sbfData 是一个辅助结构体，专门用于gob编码和解码。
type sbfData struct {
	Config  SBFConfig
	Filters []*BloomFilter
}

// ScalableBloomFilter 是一个可以自动扩容、线程安全且可持久化的布隆过滤器。
type ScalableBloomFilter struct {
	config  SBFConfig
	filters []*BloomFilter
	lock    sync.RWMutex
}

// NewScalableBloomFilter 创建一个可伸缩的布隆过滤器。
func NewScalableBloomFilter(config SBFConfig) (*ScalableBloomFilter, error) {
	if config.InitialCapacity == 0 || config.ErrorRate <= 0 || config.GrowthFactor < 1 || config.ErrorTighteningRatio <= 0 || config.ErrorTighteningRatio >= 1 {
		return nil, fmt.Errorf("无效的SBF配置参数")
	}

	// 计算第一个过滤器的误报率
	// p_i = p_0 * r^(i-1)，所以第一个过滤器的误报率是 p_0 * r^0 = p_0
	// 为了使总体误报率趋近于 ErrorRate，第一个可以稍微紧一些
	firstErrorRate := config.ErrorRate * (1 - config.ErrorTighteningRatio)
	firstFilter := NewBloomFilter(config.InitialCapacity, firstErrorRate)

	return &ScalableBloomFilter{
		config:  config,
		filters: []*BloomFilter{firstFilter},
	}, nil
}

// Add 向SBF中添加一个元素。此操作是线程安全的。
func (sbf *ScalableBloomFilter) Add(data []byte) {
	sbf.lock.Lock()
	defer sbf.lock.Unlock()

	// 获取最新的过滤器
	lastFilter := sbf.filters[len(sbf.filters)-1]

	// 如果最新的过滤器已满，则创建一个新的过滤器
	if lastFilter.isFull() {
		newCapacity := uint(float64(lastFilter.Capacity) * sbf.config.GrowthFactor)

		// 新的误报率 p_i = p_{i-1} * r
		// 我们需要从BloomFilter的 M 和 K 计算回当前的 errorRate
		currentP := math.Pow(1-math.Exp(-float64(lastFilter.K*lastFilter.ItemCount)/float64(lastFilter.M)), float64(lastFilter.K))
		newErrorRate := currentP * sbf.config.ErrorTighteningRatio

		newFilter := NewBloomFilter(newCapacity, newErrorRate)
		sbf.filters = append(sbf.filters, newFilter)
		lastFilter = newFilter
	}

	lastFilter.Add(data)
}

// Test 检查一个元素是否可能存在于SBF中。此操作是线程安全的。
func (sbf *ScalableBloomFilter) Test(data []byte) bool {
	sbf.lock.RLock()
	defer sbf.lock.RUnlock()

	// 必须检查所有过滤器，从新到旧，因为新元素总是在最新的过滤器里
	for i := len(sbf.filters) - 1; i >= 0; i-- {
		if sbf.filters[i].Test(data) {
			return true
		}
	}

	return false
}

// Len 返回SBF中的子过滤器数量。
func (sbf *ScalableBloomFilter) Len() int {
	sbf.lock.RLock()
	defer sbf.lock.RUnlock()
	return len(sbf.filters)
}

// --- 持久化功能 ---

// WriteToFile 将当前的SBF状态序列化并写入文件。
func (sbf *ScalableBloomFilter) WriteToFile(filePath string) error {
	sbf.lock.RLock()
	defer sbf.lock.RUnlock()

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)

	// 将需要持久化的数据放入 sbfData 结构体
	dataToSave := sbfData{
		Config:  sbf.config,
		Filters: sbf.filters,
	}

	if err := encoder.Encode(dataToSave); err != nil {
		return fmt.Errorf("gob编码失败: %w", err)
	}

	return nil
}

// NewScalableBloomFilterFromFile 从文件加载并创建一个新的SBF实例。
func NewScalableBloomFilterFromFile(filePath string) (*ScalableBloomFilter, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)

	var loadedData sbfData
	if err := decoder.Decode(&loadedData); err != nil {
		return nil, fmt.Errorf("gob解码失败: %w", err)
	}

	// 重新构建 SBF 实例
	sbf := &ScalableBloomFilter{
		config:  loadedData.Config,
		filters: loadedData.Filters,
	}

	return sbf, nil
}
