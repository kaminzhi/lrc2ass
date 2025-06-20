package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	// "golang.org/x/text"
)

type Syllable struct {
	Time float64
	Text string
}

func parseLine(line string) ([]Syllable, error) {
	// 用正則取出所有 <mm:ss.xx> 文字組合
	if strings.Contains(line, "<") {
		re := regexp.MustCompile(`<(\d+):(\d+\.\d+)> ?([^<]+)`)
		matches := re.FindAllStringSubmatch(line, -1)
		var result []Syllable
		for _, match := range matches {
			min, _ := strconv.Atoi(match[1])
			sec, _ := strconv.ParseFloat(match[2], 64)
			text := match[3]

			// 去除文字中可能的 [mm:ss.xx]
			cleanText := regexp.MustCompile(`\[\d+:\d+\.\d+\]`).ReplaceAllString(text, "")
			result = append(result, Syllable{
				Time: float64(min)*60 + sec,
				Text: strings.TrimSpace(cleanText),
			})
		}
		return result, nil
	}

	// 普通 LRC 格式: [mm:ss.xx] text
	re := regexp.MustCompile(`\[(\d+):(\d+\.\d+)\]\s*(.+)`)
	match := re.FindStringSubmatch(line)
	if len(match) == 4 {
		min, _ := strconv.Atoi(match[1])
		sec, _ := strconv.ParseFloat(match[2], 64)
		text := match[3]

		// 去除文字中可能的 [mm:ss.xx]
		cleanText := regexp.MustCompile(`\[\d+:\d+\.\d+\]`).ReplaceAllString(text, "")
		return []Syllable{
			{Time: float64(min)*60 + sec, Text: strings.TrimSpace(cleanText)},
		}, nil
	}

	return nil, nil
}

func formatASSTime(sec float64) string {
	h := int(sec) / 3600
	m := (int(sec) % 3600) / 60
	s := int(sec) % 60
	cs := int((sec - float64(int(sec))) * 100)
	return fmt.Sprintf("%d:%02d:%02d.%02d", h, m, s, cs)
}

// 根據前後時間與歌詞字數計算每字高亮時長(單位是0.01秒)
func buildKaraokeText(text string, start, end float64) string {
	// ASS的{\k}是以0.01秒為單位的長度
	durationCS := int((end - start) * 100) // 0.01秒為單位

	// 將歌詞依字拆分（可依需求改成依詞）
	chars := []rune(text)

	if len(chars) == 0 {
		return ""
	}

	// 平均分配每字時間長度
	perChar := durationCS / len(chars)
	if perChar == 0 {
		perChar = 1
	}

	var sb strings.Builder
	for _, ch := range chars {
		sb.WriteString(fmt.Sprintf("{\\k%d}%c", perChar, ch))
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

	type Line struct {
		Time float64
		Text string
	}

	var lines []Line

	for scanner.Scan() {
		line := scanner.Text()
		syllables, err := parseLine(line)
		if err != nil || len(syllables) == 0 {
			continue
		}

		text := ""
		for _, syl := range syllables {
			text = syl.Text
		}
		lines = append(lines, Line{Time: syllables[0].Time, Text: text})
	}

	// ASS header
	header := `[Script Info]
Title: Converted from LRC to KTV style
ScriptType: v4.00+
PlayResX: 384
PlayResY: 288
Timer: 100.0000

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,Arial,18,&H00FFFFFF,&H000000FF,&H00000000,&H64000000,1,0,0,0,100,100,0,0,1,1,0,2,10,10,10,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
`
	writer.WriteString(header)

	for i, line := range lines {
		var endTime float64
		if i+1 < len(lines) {
			endTime = lines[i+1].Time
		} else {
			endTime = line.Time + 3.0 // 最後一句延長3秒
		}

		// 將換行 \n 轉 ASS 換行 \N
		lyric := strings.ReplaceAll(line.Text, `\n`, `\N`)

		// 先拆成多行（以 \N 分割）
		parts := strings.Split(lyric, `\N`)

		var assTextParts []string
		for _, part := range parts {
			// 每一行用 buildKaraokeText 產生逐字高亮字串
			assTextParts = append(assTextParts, buildKaraokeText(part, line.Time, endTime))
		}
		// 用 ASS 換行符號連接多行
		finalText := strings.Join(assTextParts, `\N`)

		startStr := formatASSTime(line.Time)
		endStr := formatASSTime(endTime)

		dialogue := fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n", startStr, endStr, finalText)
		writer.WriteString(dialogue)
	}

	writer.Flush()

	fmt.Printf("Converted %s to %s successfully.\n", inputFile, outputFile)
}
