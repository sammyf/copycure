## COPYCURE
(License: MIT)

#### Description
Simple tool that crawls through a directory and its 
subdirectories and tries to locate and remove duplicate 
files. The comparison is done by computing a sha256 checksum

#### Requirement
You'll need to have go installed to compile the tool.

#### Installation
You can use the precompiled binaries in the release tab or ...

1. Clone this repository:
    ```bash
    git clone https://github.com/yourusername/copycure.git
    ```
2. Navigate into the directory:
    ```bash
    cd copycure
    ```
3. Build the project:
    ```bash
    go build -o copycure main.go
    ```
   
#### Usage
``` 
    ./copycure -i /path/to/your/directory -m [-m sql|mem]                  
```
replace DIRECTORY_PATH with the directory the tool should crawl through,
and method with either `mem` or `sql`.

`-m mem` stores the checksums in memory, while `-m sql` uses a temporary SQLite database to store them.
`-m sql` is default if you omit the flag.
Which option is best for you depends a lot on the speed of your RAM and your drive in your system. 

