## COPYCURE
(License: MIT)

#### Description
Simple tool that crawls through a directory and its 
subdirectories and tries to locate and remove duplicate 
files. The comparison is done by computing a sha256 checksum

#### Requirement
You'll need to have go installed if you want to compile the tool. 

### Installation
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
   
### Usage
``` 
    ./copycure -i /path/to/your/directory -m [-m sql|mem] [-y] [-e {csv}]                  
```
__parameters__

* `-m mem|sql` : stores the checksums either in RAM or in a temporary SQLite database. `-m sql` is default.
Which option is best for you depends a lot on the speed of your RAM and your drive in your system. 
* `-y` : do not ask for confirmation before deleting files. ( asking is default)
* `-e` : exclude any filepath containing any of the comma separated values. For example `-e .venv,.git,LICENSE.md` 
would ignore anything whose path include .venv, .git or LICENSE.md