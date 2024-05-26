# GO-ProxyChecker
![image](https://github.com/Pushkarup/Go-ProxyCheck/assets/148672587/fe3da9c0-061b-4f17-ae54-418bb714a0a9)


Proxy Checker is a high-performance proxy testing tool developed by Pushkar Upadhyay. It can test HTTP, SOCKS4, and SOCKS5 proxies efficiently using Go.

## Features

- **Multi-threaded Proxy Testing**: Supports up to 500 concurrent workers and 2000 concurrent connections.
- **Proxy Support**: Tests HTTP, SOCKS4, and SOCKS5 proxies.
- **Target Sites**: Randomly selects target sites for testing from a provided list.
- **Detailed Output**: Logs live and dead proxies with response times.
- **Stylized Console Output**: Color-coded and styled terminal output for better readability.

## Installation

### Prerequisites

- Go 1.15 or later
- Git

### Steps

1. **Clone the Repository**

    ```bash
    git clone https://github.com/yourusername/proxy-checker.git
    cd proxy-checker
    ```

2. **Build the Project**

    ```bash
    go build -o proxy-checker
    ```

3. **Run the Program**

    ```bash
    ./proxy-checker
    ```

## Usage

1. **Prepare Your Proxy List**

    Create a file named `proxies.txt` with one proxy per line. For example:

    ```
    192.168.1.1:8080
    192.168.1.2:1080
    ```

2. **Prepare Your Target Sites**

    Create a file named `target_sites.txt` with one target site URL per line. For example:

    ```
    http://example.com
    http://another-example.com
    ```

3. **Run the Program**

    ```bash
    ./proxy-checker
    ```

4. **Follow the On-Screen Instructions**

    The program will ask you to choose the proxy type (HTTP, SOCKS4, SOCKS5) and the proxy file name. After that, it will test the proxies and provide the results.

## Output

- The program outputs the live and dead proxies to the console.
- It also saves the live proxies to a file named `Working_<PROXY_TYPE>.txt`.

## Example

### Console Output
![image](https://github.com/Pushkarup/Go-ProxyCheck/assets/148672587/b4fbfbbb-a9b9-4a6e-9b68-ea75132b05cd)

### Output File

The live proxies will be saved in a file named `Working_HTTP.txt` (or `Working_SOCKS4.txt` or `Working_SOCKS5.txt` depending on the proxy type).

## Contributing

Contributions are welcome! Feel free to open issues or submit pull requests.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Acknowledgements

Special thanks to the Go community for providing excellent resources and libraries that made this project possible.

