package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

//go:embed autorunsc.exe tcpvcon.exe
var embeddedFiles embed.FS

func cleanupFiles(files []string) {
	for _, file := range files {
		err := os.Remove(file)
		if err != nil {
			log.Printf("Failed to delete %s: %v", file, err)
		} else {
			log.Printf("Deleted %s", file)
		}
	}
}

func main() {
	// Extract embedded executables
	extractExecutable("autorunsc.exe")
	extractExecutable("tcpvcon.exe")
	//time.Sleep(2 * time.Second)
	// Generate custom filename
	customFileName := generateCustomFilename()

	// Execute Autoruns
	autorunsFileName := customFileName + "_autoruns.csv"
	executeCommand(".\\autorunsc.exe", []string{"-accepteula", "-nobanner", "-a", "*", "-h", "-s", "-t", "-c"}, autorunsFileName)

	// Azure Blob Storage Config
	sasToken := ""
	storageAccount := ""
	containerName := ""
	// Upload Autoruns to blob Storage
	uploadToAzure(autorunsFileName, storageAccount, sasToken, containerName)

	// Execute Sysinternals netstat equivalent
	netstatFileName := customFileName + "_netstat.csv"
	executeCommand(".\\tcpvcon.exe", []string{"-accepteula", "-a", "-n", "-c"}, netstatFileName)
	uploadToAzure(netstatFileName, storageAccount, sasToken, containerName)

	// Execute Sysinfo
	sysinfoFileName := customFileName + "_sysinfo.csv"
	executeCommand("systeminfo.exe", []string{"/FO", "CSV"}, sysinfoFileName)
	uploadToAzure(sysinfoFileName, storageAccount, sasToken, containerName)

	// Cleanup files
	filesToDelete := []string{autorunsFileName, "autorunsc.exe", netstatFileName, sysinfoFileName, "tcpvcon.exe"}
	defer cleanupFiles(filesToDelete)

	//Delete self

	triage_name := "liltriage"
	exec.Command("cmd.exe", "/c", "del "+triage_name).Start()

}

func extractExecutable(name string) {
	data, err := embeddedFiles.ReadFile(name)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(name, data, 0755)
	if err != nil {
		panic(err)
	}
}

func generateCustomFilename() string {
	hostname, _ := os.Hostname()
	currentDate := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s-%s", hostname, currentDate)
}

func executeCommand(command string, args []string, outputFileName string) {
	cmd := exec.Command(command, args...)
	outputFile, err := os.Create(outputFileName)
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()

	cmd.Stdout = outputFile
	err = cmd.Run()
	if err != nil {
		panic(err)
	}
}

func uploadToAzure(blobName, storageAccount, sasToken, containerName string) {
	fileData, err := os.ReadFile(blobName)
	if err != nil {
		panic(err)
	}

	client := &http.Client{}
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s?%s", storageAccount, containerName, blobName, sasToken), strings.NewReader(string(fileData)))
	if err != nil {
		panic(err)
	}

	req.Header.Set("x-ms-blob-type", "BlockBlob")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("File uploaded successfully.")
}
