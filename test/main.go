package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/google/uuid"
)

var currentProcessing string

func main() {
	// configure the songs directory name and port
	const songsDir = "./tmp/"
	const port = 8080

	currentProcessing = ""

	// add a handler for the song files
	http.HandleFunc("/tmp/", addHeaders(http.StripPrefix("/tmp/", http.FileServer(http.Dir("./tmp")))))
	http.HandleFunc("/transCode", handleTranscode)
	http.HandleFunc("/fileInfo", handleFileInfo)

	fmt.Printf("Starting server on %v\n", port)
	log.Printf("Serving %s on HTTP port: %v\n", songsDir, port)

	// serve and log errors
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), nil))
}

// addHeaders will act as middleware to give us CORS support
func addHeaders(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	}
}

func handleTranscode(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	vfScale := r.URL.Query().Get("scale")
	ss := r.URL.Query().Get("ss")
	file := r.URL.Query().Get("file")

	// Validate parameters
	if vfScale == "" || ss == "" {
		http.Error(w, "Both vf-scale and ss parameters are required", http.StatusBadRequest)
		return
	}

	// Generate a UUID
	uuid := uuid.New().String()
	os.Mkdir("./tmp/"+uuid, os.FileMode(777))

	currentProcessing = uuid

	// Build ffmpeg command
	cmdArgs := []string{"-i", "./video/" + file, "-vf", "scale=" + vfScale, "-ss", ss, "-start_number", "0", "-hls_time", "10", "-hls_list_size", "0", "-f", "hls", "./tmp/" + uuid + "/playlist.m3u8"}
	cmd := exec.Command("ffmpeg", cmdArgs...)

	// Run the command
	//cmd.Stderr = os.Stderr
	//cmd.Stdout = os.Stdout

	cmd.Start()

	go func() {
		for {
			if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
				break
			}
			if uuid != currentProcessing {
				log.Println("More than 1 instance! killed")
				cmd.Process.Kill()
				break
			}
			time.Sleep(1 * time.Second)
			log.Println("Checking...")
		}
	}()

	for {
		if _, err := os.Stat("./tmp/" + uuid + "/playlist.m3u8"); err == nil {
			// path/to/whatever exists
			break
		}
		time.Sleep(1 * time.Second)
	}
	log.Println("OK!")
	// Respond with the generated UUID
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(uuid))
}

func handleFileInfo(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	file := r.URL.Query().Get("file")

	// Build ffmpeg command
	cmdArgs := []string{"-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", "./video/" + file}
	cmd := exec.Command("ffprobe", cmdArgs...)

	log.Println(cmd.Args)
	// Run the command
	//cmd.Stderr = os.Stderr
	//cmd.Stdout = os.Stdout

	result, _ := cmd.CombinedOutput()
	//cmd.Run()
	log.Println("RAN!")

	// Respond with the generated UUID
	w.WriteHeader(http.StatusOK)
	w.Write(result)
}
