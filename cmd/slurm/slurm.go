package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dptech-corp/dflow-extender/pkg/client"
	"github.com/dptech-corp/dflow-extender/pkg/util"
	"gopkg.in/yaml.v2"
)

type SlurmJobInfo struct {
	status string
	code   int
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: ./slurm param.yml")
	}

    content, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal("Config file not found: ", err)
	}

	config := make(util.Config)
	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		log.Fatal("Parse config error: ", err)
	}

	logChars := 0
	jobIdFile := config.GetValue("jobIdFile").(string)
	workdir := config.GetValue("workdir").(string)
	scriptFile := config.GetValue("scriptFile").(string)
	interval := config.GetValue("interval").(int)
	jobInfo := SlurmJobInfo{"PENDING", 0}

	sshClient := client.NewSSHClient(config)

	jobId, err := GetJobId(jobIdFile)
	if err != nil {
		jobId = SubmitJob(sshClient, workdir, scriptFile)
		SaveJobId(jobIdFile, jobId)
	}

	for {
		<-time.Tick(time.Duration(interval) * time.Second)
        jobInfo = GetJobInfo(sshClient, jobId)
        logChars += SyncLog(sshClient, workdir, jobId, logChars)

        if jobInfo.status == "FAILED" {
            break
        } else if jobInfo.status == "COMPLETED" {
            break
        } else if jobInfo.status == "PURGED" {
            break
        } else if jobInfo.status == "CANCELLED" {
            jobInfo.code = 1
            break
        }
	}

	os.Remove(jobIdFile)
	os.Exit(jobInfo.code)
}

func GetJobId(jobIdFile string) (int, error) {
	content, err := ioutil.ReadFile(jobIdFile)
	if err != nil {
		return 0, err
	}
	jobId, err := strconv.Atoi(string(content))
	if err != nil {
		return 0, err
	}
	return jobId, nil
}

func SaveJobId(jobIdFile string, jobId int) {
	ioutil.WriteFile(jobIdFile, []byte(strconv.Itoa(jobId)), 0644)
}

func SubmitJob(c *client.SSHClient, workdir string, scriptFile string) int {
	o, e, err := c.RunCmd("cd "+workdir+" && sbatch "+scriptFile, -1, 5) // try infinite times
	fmt.Printf(o)
	fmt.Printf(e)
	if err != nil {
		log.Fatal("Submit slurm job failed: ", err)
	}
	ws := strings.Fields(o)
	if len(ws) != 4 {
		log.Fatal("Submit slurm job failed: wrong number of fields")
	}
	jobId, err := strconv.Atoi(ws[3])
	if err != nil {
		log.Fatal("Submit slurm job failed: ", err)
	}
	return jobId
}

func SyncLog(c *client.SSHClient, workdir string, jobId int, skip int) int {
	o, e, err := c.RunCmd("dd if="+workdir+"/slurm-"+strconv.Itoa(jobId)+".out bs=1 skip="+strconv.Itoa(skip), 1, 0) // try only once
	if err == client.ErrSSHConnection {
		return 0
	} else if err != nil { // the log file does not exist
		return 0
	}
	fmt.Printf(o) // print to stdout
	bs, err := strconv.Atoi(strings.Fields(strings.Split(e, "\n")[2])[0])
	if err != nil {
		log.Fatal("Wrong format: ", err)
	}
	return bs
}

func GetJobInfo(c *client.SSHClient, jobId int) SlurmJobInfo {
	o, _, err := c.RunCmd("scontrol show job "+strconv.Itoa(jobId), 1, 0) // try only once
	if err != nil {
		return SlurmJobInfo{"UNKNOWN", 0}
	}
	i := strings.Index(o, "JobState=")
	status := strings.Fields(o[i:])[0][9:]
	i = strings.Index(o, "ExitCode=")
	j := strings.Index(o[i+9:], ":")
	code, _ := strconv.Atoi(o[i+9 : i+9+j])
	return SlurmJobInfo{status, code}
}
