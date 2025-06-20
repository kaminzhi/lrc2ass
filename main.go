package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Syllable struct {
	Time float64
	Text string
}

// 解析一行 lrc，支援逐詞格式，例如:
// [01:29.85]為妳[01:30.98]彈奏蕭邦[01:32.65]的夜曲
func parseLine(line string) ([]Syllable, error) {
	re := regexp.MustCompile(`\[(\d+):(\d+\.\d+)\]([^\[]+)`)
	matches := re.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	var sylls []Syllable
	for _, m := range matches {
		min, _ := strconv.Atoi(m[1])
		sec, _ := strconv.ParseFloat(m[2], 64)
		text := m[3]
		sylls = append(sylls, Syllable{
			Time: float64(min)*60 + sec,
			Text: strings.TrimSpace(text),
		})
	}
	return sylls, nil
}

// 格式化 ASS 時間格式: h:mm:ss.cs
func formatASSTime(sec float64) string {
	h := int(sec) / 3600
	m := (int(sec) % 3600) / 60
	s := int(sec) % 60
	cs := int((sec - float64(int(sec))) * 100)
	return fmt.Sprintf("%d:%02d:%02d.%02d", h, m, s, cs)
}

// 產生逐詞同步的 \k 樣式字幕
func buildKLine(syllables []Syllable) string {
	if len(syllables) == 0 {
		return ""
	}
	// 計算每個詞的顯示時間（毫秒）
	durations := make([]int, len(syllables))
	for i := 0; i < len(syllables)-1; i++ {
		durations[i] = int((syllables[i+1].Time - syllables[i].Time) * 100)
	}
	// 最後一個詞持續 50 個毫秒
	durations[len(syllables)-1] = 50

	var sb strings.Builder
	for i, syl := range syllables {
		// \k 代表多少百分之一秒的時間
		sb.WriteString(fmt.Sprintf("{\\k%d}%s", durations[i], syl.Text))
	}
	return sb.String()
}

// 取得純文字（不含時間碼）
func extractPlainText(syllables []Syllable) string {
	var sb strings.Builder
	for _, syl := range syllables {
		sb.WriteString(syl.Text)
	}
	return sb.String()
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: lrc2ass input.lrc output.ass")
		return
	}
	inputFile := os.Args[1]
	outputFile := os.Args[2]

	file, err := os.Open(inputFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Open input file error:", err)
		return
	}
	defer file.Close()

	outf, err := os.Create(outputFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Create output file error:", err)
		return
	}
	defer outf.Close()

	writer := bufio.NewWriter(outf)
	scanner := bufio.NewScanner(file)

	var allLines [][]Syllable
	for scanner.Scan() {
		line := scanner.Text()
		sylls, err := parseLine(line)
		if err != nil || len(sylls) == 0 {
			continue
		}
		allLines = append(allLines, sylls)
	}

	// ASS header
	header := `[Script Info]
Title: Converted from LRC (KTV style)
ScriptType: v4.00+
PlayResX: 384
PlayResY: 288
Timer: 100.0000

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,Arial,20,&H00FFFFFF,&H000000FF,&H00000000,&H64000000,0,0,0,0,100,100,0,0,1,1,0,2,10,10,10,1
Style: NextLine,Arial,18,&H00AAAAAA,&H000000FF,&H00000000,&H64000000,0,0,0,0,100,100,0,0,1,1,0,2,10,10,40,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
`
	writer.WriteString(header)

	for i := 0; i < len(allLines); i++ {
		line := allLines[i]
		start := line[0].Time
		var end float64
		if i+1 < len(allLines) {
			end = allLines[i+1][0].Time
		} else {
			end = start + 3 // fallback duration
		}

		// 主行：逐詞同步（帶 \k）
		writer.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n",
			formatASSTime(start), formatASSTime(end), buildKLine(line)))

		// 下一行：純文字，顯示在下一行，無逐詞效果
		if i+1 < len(allLines) {
			nextText := extractPlainText(allLines[i+1])
			writer.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,NextLine,,0,0,0,,%s\n",
				formatASSTime(start), formatASSTime(end), nextText))
		}
	}

	writer.Flush()
	fmt.Printf("Converted %s to %s successfully.\n", inputFile, outputFile)
}
