package utils

import (
	"math"
	"strings"
	"unicode"
)

// RemoveDuplicatesStrings 字符串列表去重
func RemoveDuplicatesStrings(ss []string, threshold float64) []string {
	uniqueStrings := make([]string, 0, len(ss))

	for _, s := range ss {
		isUnique := true
		for _, unique := range uniqueStrings {
			similarity := cosineSimilarity(s, unique)
			if similarity >= threshold { // 相似度高于threshold
				isUnique = false
				break
			}

		}

		if isUnique {
			uniqueStrings = append(uniqueStrings, s)
		}
	}

	return uniqueStrings
}

// IsUniqueStrings 列表中是否存在某一项
func IsUniqueStrings(ss []string, sub string, threshold float64) bool {
	for _, s := range ss {
		similarity := cosineSimilarity(s, sub)
		if similarity >= threshold { // 相似度高于threshold
			return false
		}
	}

	return true
}

// cosineSimilarity 计算两个字符串的余弦相似度
func cosineSimilarity(s1, s2 string) float64 {
	words1 := tokenize(s1)
	words2 := tokenize(s2)

	tf1 := termFrequency(words1)
	tf2 := termFrequency(words2)

	// 计算余弦相似度
	dotProduct := 0.0
	norm1 := 0.0
	norm2 := 0.0

	for word, count1 := range tf1 {
		count2 := tf2[word]
		dotProduct += float64(count1 * count2)
		norm1 += float64(count1 * count1)
	}

	for _, count2 := range tf2 {
		norm2 += float64(count2 * count2)
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// termFrequency 文本词频统计
func termFrequency(words []string) map[string]int {
	tf := make(map[string]int)
	for _, word := range words {
		tf[word]++
	}
	return tf
}

// tokenize 文本分词
func tokenize(s string) []string {
	s = strings.ToLower(s)
	s = strings.TrimFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	return strings.Fields(s)
}
