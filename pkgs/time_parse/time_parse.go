package time_parse

import (
	"fmt"
	"math"
	"net/http"
	"time"
)

var (
	cst *time.Location
)

// CSTLayout China Standard Time Layout
const CSTLayout = "2006-01-02 15:04:05"

func init() {
	var err error
	if cst, err = time.LoadLocation("Local"); err != nil {
		panic(err)
	}
}

// RFC3339ToCSTLayout convert rfc3339 value to china standard time layout
// 2020-11-08T08:18:46+08:00 => 2020-11-08 08:18:46
func RFC3339ToCSTLayout(value string) (string, error) {
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return "", err
	}

	return ts.In(cst).Format(CSTLayout), nil
}

// CSTLayoutString 格式化时间
// 返回 "2006-01-02 15:04:05" 格式的时间
func CSTLayoutString() string {
	ts := time.Now()
	return ts.In(cst).Format(CSTLayout)
}

// 返回n月n日前时间 20060102
func GetOldTime(monthsAgo, daysAgo int) string {
	now := time.Now()
	ago := now.AddDate(0, -monthsAgo, -daysAgo)
	return ago.Format("20060102")
}

// //返回n月n日前0时 2006-01-02 15:04:05
func GetOldZeroTime(monthsAgo, daysAgo int) string {
	now := time.Now()
	ago := now.AddDate(0, -monthsAgo, -daysAgo)
	zeroHour := time.Date(ago.Year(), ago.Month(), ago.Day(), 0, 0, 0, 0, ago.Location())
	return zeroHour.Format(CSTLayout)
}

// ParseCSTInLocation 格式化时间
func ParseCSTInLocation(date string) (time.Time, error) {
	return time.ParseInLocation(CSTLayout, date, cst)
}

// CSTLayoutStringToUnix 返回 unix 时间戳
// 2020-01-24 21:11:11 => 1579871471
func CSTLayoutStringToUnix(cstLayoutString string) (int64, error) {
	stamp, err := time.ParseInLocation(CSTLayout, cstLayoutString, cst)
	if err != nil {
		return 0, err
	}
	return stamp.Unix(), nil
}

// GMTLayoutString 格式化时间
// 返回 "Mon, 02 Jan 2006 15:04:05 GMT" 格式的时间
func GMTLayoutString() string {
	return time.Now().In(cst).Format(http.TimeFormat)
}

// ParseGMTInLocation 格式化时间
func ParseGMTInLocation(date string) (time.Time, error) {
	return time.ParseInLocation(http.TimeFormat, date, cst)
}

// 2006-01-02T15:04:05.999+0800
func ParseTZInLocation(date string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02T15:04:05.999+0800", date, cst)
}

// 2022-03-31T08:07:51.795013095Z
func ParseTZ2InLocation(date string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02T15:04:05.999999999Z", date, cst)
}

// "2023-08-08T09:14:52.000Z"
func ParseTZ3InLocation(date string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02T15:04:05.000Z", date, cst)
}

// SubInLocation 计算时间差
func SubInLocation(ts time.Time) float64 {
	return math.Abs(time.Now().In(cst).Sub(ts).Seconds())
}

// "2006-01-02T15:04:05.999+0800"
func SelfLayoutTime1() string {
	return time.Now().Format("2006-01-02T15:04:05.999+0800")
}

// 20151212T121212Z
func SelfLayoutTime2() string {
	return time.Now().Format("20060102T150405Z")
}

// 20060102150405
func SelfLayoutTime3() string {
	return time.Now().Format("20060102150405")
}

func GetUnixS() string {
	return fmt.Sprint(time.Now().Unix())
}

// 1616573517364
func GetUnixMS() string {
	return fmt.Sprint(time.Now().UnixNano() / 1e6)
}

func GetUnixUS() string {
	return fmt.Sprint(time.Now().UnixNano() / 1e3)
}

func GetUnixNS() string {
	return fmt.Sprint(time.Now().UnixNano())
}
