package main

import (
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

/* Globals */

var appName string

var logFile *os.File
var err error

// Interface(s)
var output string

var ifacePwm bool
var auxPwm bool

var src []byte
var dest []byte

var srcPwm []byte

var iface *net.Interface
var sock int
var addr syscall.SockaddrLinklayer

var auxIface *net.Interface
var auxSock int
var auxAddr syscall.SockaddrLinklayer

// Environment
var locale *time.Location
var assets string

// Display
type Daytime struct {
	val bool
}

type Brightness struct {
	val int
}

var width int
var height int

var fps int

var daytime Daytime
var brightness Brightness

var maskW int
var maskH int
var maskX int
var maskY int

var multiplier float64

// Playback
var defaultSlide string
var systemFallback string
var duration int

// State
var manifest *register

/* Initialisation */

func setup() {

	// Set application name
	appName = "ledlight"

	// Setup logging
	logDir := "/var/log/" + appName
	err = os.MkdirAll(logDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	logPath := logDir + "/" + today() + ".log"
	logFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(logFile)

	log.Println("[INFO] Initialising " + cases.Title(language.English, cases.Compact).String(appName) + " service")

	// Load environment variables
	exe, err := os.Executable()
	if err != nil {
		log.Fatal("[FATAL] ", err)
	}

	err = godotenv.Load(filepath.Dir(exe) + "/.env")
	if err != nil {
		log.Fatal("[FATAL] ", err)
	}

	// Set globals
	output = env("DEVICE_OUTPUT", "single")
	if output != "single" && output != "dual" {
		log.Fatal("[FATAL] DEVICE_OUTPUT must be single or dual")
	}

	ifacePwm = mustBool("DEVICE_IFACE_PWM", "false")
	auxPwm = mustBool("DEVICE_AUX_PWM", "false")

	iface, err = net.InterfaceByName(env("DEVICE_IFACE", ""))
	if err != nil {
		log.Fatal("[FATAL] ", err)
	}

	src = []byte{0x22, 0x22, 0x33, 0x33, 0x55, 0x66}  // MAC address of source (22:22:33:33:55:66)
	dest = []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66} // Mac address of destination (11:22:33:44:55:66)

	srcPwm = []byte{0x22, 0x22, 0x33, 0x44, 0x55, 0x66} // MAC address of source (22:22:33:44:55:66), for PWM cards

	sock, err = syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, syscall.ETH_P_ALL)
	if err != nil {
		log.Fatal("[FATAL] ", err)
	}

	addr = syscall.SockaddrLinklayer{
		Protocol: syscall.ETH_P_ALL,
		Ifindex:  iface.Index,
		Halen:    6,
		Addr: [8]uint8{
			dest[0], dest[1], dest[2], dest[3], dest[4], dest[5],
		},
	}

	if output == "dual" {
		auxSock, err = syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, syscall.ETH_P_ALL)
		if err != nil {
			log.Fatal("[FATAL] ", err)
		}

		auxIface, err = net.InterfaceByName(env("DEVICE_AUX_IFACE", ""))
		if err != nil {
			log.Fatal("[FATAL] ", err)
		}

		auxAddr = syscall.SockaddrLinklayer{
			Protocol: syscall.ETH_P_ALL,
			Ifindex:  auxIface.Index,
			Halen:    6,
			Addr: [8]uint8{
				dest[0], dest[1], dest[2], dest[3], dest[4], dest[5],
			},
		}
	}

	locale, err = time.LoadLocation(env("LOCALE", ""))
	if err != nil {
		log.Fatal("[FATAL] ", err)
	}

	assets = env("ASSET_PATH", "")
	if assets == "" {
		log.Fatal("[FATAL] ASSET_PATH is required")
	}

	width = mustPositiveInt("DISPLAY_WIDTH", "0")
	height = mustPositiveInt("DISPLAY_HEIGHT", "0")

	fps = mustPositiveInt("FRAME_RATE", "60")

	daytime.update()
	brightness.update()

	maskW = mustNonNegativeInt("SUBMASK_WIDTH", "0")
	maskH = mustNonNegativeInt("SUBMASK_HEIGHT", "0")
	maskX = mustNonNegativeInt("SUBMASK_POS_X", "0")
	maskY = mustNonNegativeInt("SUBMASK_POS_Y", "0")

	multiplier = mustFloat("SUBMASK_MULTIPLIER", "1")
	if multiplier < 0 {
		log.Fatal("[FATAL] SUBMASK_MULTIPLIER must be 0 or greater")
	}

	if output == "dual" {
		if maskW <= 0 || maskH <= 0 {
			log.Fatal("[FATAL] SUBMASK_WIDTH and SUBMASK_HEIGHT must be greater than 0 for dual output")
		}

		if maskX+maskW > width || maskY+maskH > height {
			log.Fatal("[FATAL] SUBMASK dimensions must fit inside DISPLAY dimensions")
		}
	}

	defaultSlide = env("DEFAULT_SLIDE", "")
	systemFallback = "/opt/ledlight/fallback.png"
	duration = mustPositiveInt("SLIDE_DURATION", "5")

	manifest = &register{}
}

/* Helpers */

// Config Checks
func mustBool(key string, fallback string) bool {
	val, err := strconv.ParseBool(env(key, fallback))
	if err != nil {
		log.Fatal("[FATAL] ", key, " must be a boolean")
	}

	return val
}

func mustPositiveInt(key string, fallback string) int {
	val := mustInt(key, fallback)
	if val <= 0 {
		log.Fatal("[FATAL] ", key, " must be greater than 0")
	}

	return val
}

func mustNonNegativeInt(key string, fallback string) int {
	val := mustInt(key, fallback)
	if val < 0 {
		log.Fatal("[FATAL] ", key, " must be 0 or greater")
	}

	return val
}

func mustPercent(key string, fallback string) int {
	val := mustInt(key, fallback)
	if val < 0 || val > 100 {
		log.Fatal("[FATAL] ", key, " must be between 0 and 100")
	}

	return val
}

func mustInt(key string, fallback string) int {
	val, err := strconv.Atoi(env(key, fallback))
	if err != nil {
		log.Fatal("[FATAL] ", key, " must be an integer")
	}

	return val
}

func mustFloat(key string, fallback string) float64 {
	val, err := strconv.ParseFloat(env(key, fallback), 64)
	if err != nil {
		log.Fatal("[FATAL] ", key, " must be a number")
	}

	return val
}

func mustClock(key string, fallback string) time.Time {
	val := env(key, fallback)

	for _, layout := range []string{"15:04", "3:04"} {
		parsed, err := time.ParseInLocation(layout, val, locale)
		if err == nil {
			return parsed
		}
	}

	log.Fatal("[FATAL] ", key, " must use HH:MM format")
	return time.Time{}
}

// Environment Variables
func env(key string, fallback string) string {
	val, exists := os.LookupEnv(key)

	if exists {
		return val
	}

	return fallback
}

// Daytime Flag
func (d *Daytime) check() bool {
	return d.val
}

func (d *Daytime) update() {
	now := time.Now().In(locale)
	current := time.Date(0, 1, 1, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), locale)

	dayStart := mustClock("DAY_START", "06:00")
	dayEnd := mustClock("DAY_END", "18:00")

	prev := d.val

	if dayStart.Before(dayEnd) {
		d.val = current.After(dayStart) && current.Before(dayEnd)
	} else {
		d.val = current.After(dayStart) || current.Before(dayEnd)
	}

	if d.val != prev {
		if !d.val {
			log.Println("[INFO] Entering night mode")
		} else {
			log.Println("[INFO] Entering day mode")
		}
	}
}

// Current Brightness
func (b *Brightness) get() int {
	return b.val
}

func (b *Brightness) rgb() int {
	brightness := gamma(65535 * (float64(b.val) / 100))
	return int(i32tob(uint32(brightness)))
}

func (b *Brightness) lin() int {
	brightness := 65535 * (float64(b.val) / 100)
	return int(i32tob(uint32(brightness)))
}

func (b *Brightness) update() {
	day := mustPercent("BRIGHTNESS_DAY", "100")
	night := mustPercent("BRIGHTNESS_NIGHT", "50")

	prev := b.val

	if daytime.check() {
		b.val = day
	} else {
		b.val = night
	}

	if b.val != prev {
		log.Println("[INFO] Adjusting brightness to " + strconv.Itoa(b.val) + "%")
	}
}
