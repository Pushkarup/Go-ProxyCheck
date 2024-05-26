package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

// Foreground represents custom foreground colors for console output.
type Foreground struct {
	Black string
	Red   string
	Green string
	Cyan  string
	White string
}

// Fore is an instance of the Foreground struct containing custom foreground colors.
var Fore = Foreground{
	Black: "\033[30m",
	Red:   "\033[31m",
	Green: "\033[32m",
	Cyan:  "\033[36m",
	White: "\033[37m",
}

// Styling represents custom text styles for console output.
type Styling struct {
	Italic    string
	Blink     string
	Reset_all string
}

// Style is an instance of the Styling struct containing custom text styles.
var Style = Styling{
	Italic:    "\033[3m",
	Blink:     "\033[5m",
	Reset_all: "\033[0m",
}

const (
	maxWorkers               = 500              // maxWorkers specifies the maximum number of worker goroutines for concurrent proxy testing.
	maxConcurrentConnections = 2000             // maxConcurrentConnections specifies the maximum number of concurrent connections allowed for proxy testing.
	connectionTimeout        = 10 * time.Second // connectionTimeout specifies the timeout duration for establishing connections during proxy testing.
)

var targetSites []string

// ///////////////////////////////////////////////////////////////////////////////////////////////
// main is the entry point of the program. It orchestrates the proxy testing process, including reading target sites, testing proxies, and generating a summary report.
//
// Parameters:
//
//	None.
//
// Returns:
//
//	None.
//
// Note:
//
//	This function clears the console, loads target sites from a file, displays a logo, and prompts the user to choose a proxy type (HTTP/HTTPS, SOCKS4, or SOCKS5).
//	It then reads proxy addresses from a file, tests each proxy concurrently, and writes the working proxies to an output file.
//	Finally, it generates a summary of the proxy testing process, including the number of target sites, total lines read, and the number of live and dead proxies.
//	The program waits for 15 seconds before exiting.
func main() {
	clearConsole()
	loadTargetSites("target_sites.txt")
	logo()

	fmt.Println("")
	fmt.Println("                      █▀ █▀▀ █░░ █▀▀ █▀▀ ▀█▀  " + Fore.Red + "  █▀█ █▀█ █▀█ ▀▄▀ █▄█" + Style.Reset_all)
	fmt.Println("                      ▄█ ██▄ █▄▄ ██▄ █▄▄ ░█░  " + Fore.Red + "  █▀▀ █▀▄ █▄█ █░█ ░█░" + Style.Reset_all)
	fmt.Println("")
	fmt.Println("                              [1] HTTP/HTTPS")
	fmt.Println("                              [2] SOCKS4")
	fmt.Println("                              [3] SOCKS5")

	choice := getIntInput(Fore.Cyan + "[ENTER]" + Fore.White + " YOUR CHOICE: ")
	fmt.Printf("\n")
	fileName := getInput(Fore.Cyan + "[ENTER]" + Fore.White + " PROXY FILE: ")
	linecount, _ := CountLines(fileName)

	proxies, err := readProxiesFromFile(fileName)
	if err != nil {
		fmt.Printf(Fore.Red + "[ERROR]" + Fore.White + " reading proxies from file ")
		return
	}

	if len(proxies) == 0 {
		fmt.Printf(Fore.Red + "[ERROR]" + Fore.White + " No proxies found in the file ")
		return
	}
	var proxyType string
	switch choice {
	case 1:
		proxyType = "HTTP"
	case 2:
		proxyType = "SOCKS4"
	case 3:
		proxyType = "SOCKS5"
	default:
		fmt.Printf(Fore.Red + "[ERROR]" + Fore.White + " Invalid Choice")
		return
	}

	var workingProxies []string

	semaphore := make(chan struct{}, maxConcurrentConnections)
	var wg sync.WaitGroup
	var mu sync.Mutex

	workers := make(chan struct{}, maxWorkers)
	for _, proxy := range proxies {
		workers <- struct{}{}
		wg.Add(1)
		go func(proxy string) {
			defer wg.Done()
			defer func() { <-workers }()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			status, result := testProxy(proxy, proxyType)
			if status {
				mu.Lock()
				fmt.Printf("%s[+]  %s%s         | %sLIVE     | %s%s%s\n", Fore.Green, Fore.White, proxy, Fore.Green, Fore.White, result, Style.Reset_all)
				workingProxies = append(workingProxies, fmt.Sprintf("%s   |  %s", proxy, result))
				mu.Unlock()
			} else {
				fmt.Printf("%s[~] %s%s         | %sDEAD%s     |  \n", Fore.Red, Fore.White, proxy, Fore.Red, Style.Reset_all)
			}
		}(proxy)
	}

	wg.Wait()

	if len(workingProxies) == 0 {
		fmt.Printf("No working %s proxies found.\n", proxyType)
		return
	}

	fileNameOutput := fmt.Sprintf("Working_%s.txt", proxyType)
	writeFile(fileNameOutput, workingProxies)
	Summary(linecount, len(workingProxies))
	time.Sleep(15 * time.Second)
}

// ///////////////////////////////////////////////////////////////////////////////////////////////
// getInput prompts the user with the provided string and reads a line of input from the console.
// It returns the input as a string.
//
// Parameters:
//   - prompt: A string that will be displayed to the user as the prompt message.
//
// Returns:
//   - A string containing the input entered by the user.
//

func getInput(prompt string) string {
	var input string
	fmt.Print(prompt)
	fmt.Scanln(&input)
	return input
}

////////////////////////////////////////////////////////////////////////////////////////////////
// getIntInput prompts the user with the provided string and reads an integer input from the console.
// It returns the input as an integer.
//
// Parameters:
//   - prompt: A string that will be displayed to the user as the prompt message.
//
// Returns:
//   - An integer containing the input entered by the user.
//

func getIntInput(prompt string) int {
	var input int
	fmt.Print(prompt)
	fmt.Scanln(&input)
	return input
}

// ///////////////////////////////////////////////////////////////////////////////////////////////
// readProxiesFromFile reads a list of proxy addresses from a file and returns them as a slice of strings.
// Each line in the file should contain one proxy address. Leading and trailing whitespace will be trimmed.
//
// Parameters:
//   - fileName: The name of the file containing the list of proxy addresses.
//
// Returns:
//   - A slice of strings, each representing a proxy address.
//   - An error if any issues occur while opening the file or reading its contents.
//

func readProxiesFromFile(fileName string) ([]string, error) {
	var proxies []string

	file, err := os.Open(fileName)
	if err != nil {
		return proxies, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		proxy := strings.TrimSpace(scanner.Text())
		proxies = append(proxies, proxy)
	}

	if err := scanner.Err(); err != nil {
		return proxies, err
	}

	return proxies, nil
}

// ///////////////////////////////////////////////////////////////////////////////////////////////
// loadTargetSites reads a list of target sites from a file and appends them to the global targetSites slice.
// Each line in the file should contain one target site URL. Leading and trailing whitespace will be trimmed.
// If an error occurs while opening or reading the file, it prints an error message.
//
// Parameters:
//   - fileName: The name of the file containing the list of target sites.
//
// Returns:
//   None.
//
// Note:
//   This function assumes that the variable targetSites is declared as a global slice of strings:
//   var targetSites []string
//

func loadTargetSites(fileName string) {
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Printf(Fore.Red+"[ERROR]"+Fore.White+" Can't Read Target Sites : ", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		targetSite := strings.TrimSpace(scanner.Text())
		targetSites = append(targetSites, targetSite)
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf(Fore.Red+"[ERROR]"+Fore.White+" Can't Scan target sites: ", err)
		return
	}
}

// ///////////////////////////////////////////////////////////////////////////////////////////////
// getRandomTargetSite selects and returns a random target site from the global targetSites slice.
// It seeds the random number generator with the current Unix timestamp to ensure randomness.
//
// Returns:
//   - A string representing a randomly selected target site from the targetSites slice.
//
// Note:
//   This function assumes that the variable targetSites is declared as a global slice of strings and is non-empty:
//   var targetSites []string
//
//   Ensure that targetSites is populated before calling this function to avoid a runtime panic due to accessing an empty slice.

func getRandomTargetSite() string {
	rand.Seed(time.Now().UnixNano())
	index := rand.Intn(len(targetSites))
	return targetSites[index]
}

//////////////////////////////////////////////////////////////////////////////////////////////////
// testProxy tests the provided proxy with a randomly selected target site URL.
// It selects a random target site from the global targetSites slice and tests the proxy against it.
// The proxy type determines which specific test function (testHTTPProxy, testSOCKS4Proxy, or testSOCKS5Proxy) to call.
//
// Parameters:
//   - proxyAddress: The address of the proxy to test.
//   - proxyType: The type of the proxy (e.g., "HTTP", "SOCKS4", "SOCKS5").
//
// Returns:
//   - A boolean value indicating whether the proxy test was successful (true) or not (false).
//   - A string containing additional information or an error message, if any.
//
// Note:
//   This function assumes that the global targetSites slice is populated with target site URLs.
//   Ensure that targetSites is populated before calling this function to avoid runtime errors.

func testProxy(proxyAddress, proxyType string) (bool, string) {
	targetURL := getRandomTargetSite()

	switch proxyType {
	case "HTTP":
		return testHTTPProxy(proxyAddress, targetURL)
	case "SOCKS4":
		return testSOCKS4Proxy(proxyAddress, targetURL)
	case "SOCKS5":
		return testSOCKS5Proxy(proxyAddress, targetURL)
	default:
		return false, ""
	}
}

// ////////////////////////////////////////////////////////////////////////////////////////////////
// testHTTPProxy tests the provided HTTP proxy with the specified target URL.
// It sends an HTTP GET request to the target URL using the provided proxy address.
// If the request succeeds (HTTP status code 200 OK), it measures the response time and returns true along with the response time in milliseconds.
// If the request fails or the response status code is not 200 OK, it returns false.
//
// Parameters:
//   - proxyAddress: The address of the HTTP proxy to test.
//   - targetURL: The URL of the target site to test the proxy against.
//
// Returns:
//   - A boolean value indicating whether the HTTP proxy test was successful (true) or not (false).
//   - A string containing the response time in milliseconds if the test was successful, or an empty string if unsuccessful.

func testHTTPProxy(proxyAddress, targetURL string) (bool, string) {
	start := time.Now()
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse("http://" + proxyAddress)
			},
		},
		Timeout: connectionTimeout,
	}

	resp, err := client.Get(targetURL)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		duration := time.Since(start)
		return true, fmt.Sprintf("%d ms", duration.Milliseconds())
	}
	return false, ""
}

// ////////////////////////////////////////////////////////////////////////////////////////////////
// testSOCKS4Proxy tests the provided SOCKS4 proxy with the specified target URL.
// It establishes a TCP connection to the SOCKS4 proxy and sends a SOCKS4 request to connect to the target URL.
// If the SOCKS4 request is successful (indicated by the response code), it measures the connection establishment time and returns true along with the response time in milliseconds.
// If the SOCKS4 request fails or the response code indicates failure, it returns false.
//
// Parameters:
//   - proxyAddress: The address of the SOCKS4 proxy to test.
//   - targetURL: The URL of the target site to test the proxy against.
//
// Returns:
//   - A boolean value indicating whether the SOCKS4 proxy test was successful (true) or not (false).
//   - A string containing the response time in milliseconds if the test was successful, or an empty string if unsuccessful.

func testSOCKS4Proxy(proxyAddress, targetURL string) (bool, string) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", proxyAddress, connectionTimeout)
	if err != nil {
		return false, ""
	}
	defer conn.Close()

	request := []byte{0x04, 0x01, 0x00, 0x50}
	request = append(request, []byte(targetURL)...)
	request = append(request, 0x00)

	_, err = conn.Write(request)
	if err != nil {
		return false, ""
	}

	response := make([]byte, 8)
	_, err = conn.Read(response)
	if err != nil {
		return false, ""
	}
	if response[0] != 0x00 || response[1] != 0x5a {
		return false, ""
	}
	duration := time.Since(start)
	return true, fmt.Sprintf("%d ms", duration.Milliseconds())
}

// ////////////////////////////////////////////////////////////////////////////////////////////////
// testSOCKS5Proxy tests the provided SOCKS5 proxy with the specified target URL.
// It establishes a SOCKS5 proxy connection to the target URL using the provided SOCKS5 proxy address.
// If the SOCKS5 proxy connection is successful, it measures the connection establishment time and returns true along with the response time in milliseconds.
// If the SOCKS5 proxy connection fails, it returns false.
//
// Parameters:
//   - proxyAddress: The address of the SOCKS5 proxy to test.
//   - targetURL: The URL of the target site to test the proxy against.
//
// Returns:
//   - A boolean value indicating whether the SOCKS5 proxy test was successful (true) or not (false).
//   - A string containing the response time in milliseconds if the test was successful, or an empty string if unsuccessful.

func testSOCKS5Proxy(proxyAddress, targetURL string) (bool, string) {
	start := time.Now()
	dialer, err := proxy.SOCKS5("tcp", proxyAddress, nil, &net.Dialer{Timeout: connectionTimeout})
	if err != nil {
		return false, ""
	}

	conn, err := dialer.Dial("tcp", targetURL)
	if err == nil {
		conn.Close()
		duration := time.Since(start)
		return true, fmt.Sprintf("%d ms", duration.Milliseconds())
	}
	return false, ""
}

// ////////////////////////////////////////////////////////////////////////////////////////////////
// writeFile writes the list of proxies to a file with the specified file name.
// Each proxy address is written to a separate line in the file.
//
// Parameters:
//   - fileName: The name of the file to write the proxies to.
//   - proxies: A slice containing the proxy addresses to write to the file.
//
// Returns:
//   None.
//
// Note:
//   This function creates a new file if it does not exist, or overwrites the existing file if it does.

func writeFile(fileName string, proxies []string) {
	file, err := os.Create(fileName)
	if err != nil {
		fmt.Printf(Fore.Red+"[ERROR]"+Fore.White+" Failed to create file:", err)

		return
	}
	defer file.Close()

	for _, proxy := range proxies {
		_, err := file.WriteString(proxy + "\n")
		if err != nil {
			fmt.Printf(Fore.Red+"[ERROR]"+Fore.White+" Failed to create file:", err)
			return
		}
	}
}

// ////////////////////////////////////////////////////////////////////////////////////////////////
// logo prints a stylized logo to the standard output.
// The logo includes text and styling to enhance its appearance.
//
// Parameters:
//   None.
//
// Returns:
//   None.
//
// Note:
//   This function is designed to be used for visual branding or decoration purposes.

func logo() {
	fmt.Printf("                  ▒█▀▀█ ▒█▀▀█ ▒█▀▀▀█ ▀▄▒▄▀ ▒█░░▒█   %v▒█▀▀█ ▒█░▒█ ▒█▀▀▀ ▒█▀▀█ ▒█░▄▀ ▒█▀▀▀ ▒█▀▀█ %v\n", Style.Blink+Fore.Red, Style.Reset_all)
	fmt.Printf("                  ▒█▄▄█ ▒█▄▄▀ ▒█░░▒█ ░▒█░░ ▒█▄▄▄█   %v▒█░░░ ▒█▀▀█ ▒█▀▀▀ ▒█░░░ ▒█▀▄░ ▒█▀▀▀ ▒█▄▄▀ %v\n", Style.Blink+Fore.Red, Style.Reset_all)
	fmt.Printf("                  ▒█░░░ ▒█░▒█ ▒█▄▄▄█ ▄▀▒▀▄ ░░▒█░░   %v▒█▄▄█ ▒█░▒█ ▒█▄▄▄ ▒█▄▄█ ▒█░▒█ ▒█▄▄▄ ▒█░▒█ %v\n", Style.Blink+Fore.Red, Style.Reset_all)
	fmt.Printf("                                         %vA PROGRAM BY PUSHKAR UPADHYAY                           %v  \n", Style.Italic+Fore.Cyan, Style.Reset_all)
	fmt.Println("")
}

// ////////////////////////////////////////////////////////////////////////////////////////////////
// clearConsole clears the contents of the console window.
// It executes the appropriate command for clearing the console based on the operating system.
//
// Parameters:
//   None.
//
// Returns:
//   None.
//
//
// Note:
//   This function is intended to clear the console window for improved readability or visual presentation.

func clearConsole() {
	cmd := exec.Command("cmd", "/c", "cls")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

// ////////////////////////////////////////////////////////////////////////////////////////////////
// CountLines reads the content of the file specified by the filename parameter and counts the number of lines in it.
// It returns the number of lines found in the file.
//
// Parameters:
//   - filename: The name of the file to read and count lines from.
//
// Returns:
//   - An integer representing the number of lines in the file.
//   - An error if any issues occur while reading the file.

func CountLines(filename string) (int, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return 0, err
	}
	lines := strings.Split(string(data), "\n")
	return len(lines), nil
}

// ////////////////////////////////////////////////////////////////////////////////////////////////
// Summary generates a summary of the proxy testing process including the number of target sites, total lines read, and the number of live and dead proxies.
//
// Parameters:
//   - lines: The total number of lines read from the file containing proxies.
//   - proxies: The total number of live proxies.
//
// Returns:
//   None.
//
// Note:
//   This function prints a summary of the proxy testing process including the number of target sites, total lines read, and the number of live and dead proxies.
//   It also includes a software attribution line mentioning the author.

func Summary(lines int, proxies int) {

	sites, _ := CountLines("target_sites.txt")

	fmt.Printf("\n            █▀ █░█ █▀▄▀█ █▀▄▀█ ▄▀█ █▀█ █▄█ ____________\n")
	fmt.Printf("|           ▄█ █▄█ █░▀░█ █░▀░█ █▀█ █▀▄ ░█░             \n\n")
	fmt.Printf("|             TARGET SITES = %v\n", sites)
	fmt.Printf("|             TOTAL LINES  = %v\n", lines)
	fmt.Printf("|             TOTAL LIVE   = %v\n", proxies)
	fmt.Printf("|             TOTAL DEAD   = %v\n", lines-proxies)
	fmt.Printf("|  \n")
	fmt.Printf("|             A SOFTWARE BY PUSHKAR UPADHYAY")

}
