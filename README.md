# :memo: lx

`lx` is a tool for running commands and filtering their output. It is similar to the `docker logs` command but with additional filtering capabilities (WIP).

## Installation

To install this project, you need to have Go installed. Then, you can install the project by running the following command:

```sh
go install github.com/Geun-Oh/lx
```

## Usage

To use lx to run a command and filter its output, you can use the following command:

```sh
lx <filter-text> <command> [command-args...]
```

For example, to filter the output of the echo command, you can use:

```sh
lx LOG echo "LOG: HELLO WORLD"
```

This command will filter the output and display only the lines containing the text hello.

## Contributing
Contributions are welcome! You can contribute by reporting bugs, requesting features, or submitting pull requests.

## License
This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for more details.
