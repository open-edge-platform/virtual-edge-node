# Script Name: Show_vms_data.ps1
# Description: Open multiple terminals using Windows Terminal.
# 				Run it from windows
# Version: 1.0
# Author: Nabendu Maiti
# Date: 2024-11-01

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

## Before run, create public key in local system and transfer to remote system, to do passwordless login
##
## ssh-keygen -t rsa -b 4096 -C "your_email@example.com"
## ssh-copy-id -i ~/.ssh/id_ecdsa.pub intel@10.49.76.112
## ssh-copy-id -i ~/.ssh/id_ecdsa.pub intel@10.49.76.113
## ssh-copy-id -i ~/.ssh/id_ecdsa.pub intel@10.49.76.157
## ssh-copy-id -i ~/.ssh/id_ecdsa.pub hspe@10.49.76.159
## ssh-copy-id -i ~/.ssh/id_ecdsa.pub user@10.49.76.160
## ssh-copy-id -i ~/.ssh/id_ecdsa.pub hspe@10.49.76.203

#### Enable Powershell run script allow
#### 1) Open PowerShell as Administrator: Win +x
#### 2) Set-ExecutionPolicy RemoteSigned
#### 3) Get-ExecutionPolicy


# Define the SSH connections for each server
$servers = @(
    @{ User = "intel"; Host = "10.49.76.112" },
    @{ User = "intel"; Host = "10.49.76.113" },
    @{ User = "intel"; Host = "10.49.76.157" },
    @{ User = "hspe"; Host = "10.49.76.159" },
    @{ User = "user"; Host = "10.49.76.160" },
    @{ User = "hspe"; Host = "10.49.76.203" }
)

# Cleanup previously running background jobs
Get-Job | Where-Object { $_.State -eq 'Running' } | Stop-Job
Get-Job | Remove-Job

# Create the temporary log directory if it doesn't exist
if (-Not (Test-Path -Path "C:\temp")) {
    New-Item -ItemType Directory -Path "C:\temp"
}

# Remove older log files before collecting new logs
Get-ChildItem -Path "C:\temp\logfile_*.log" -ErrorAction SilentlyContinue | Remove-Item -Force
if (Test-Path -Path "C:\temp\merged_log.log") {
    Remove-Item -Path "C:\temp\merged_log.log" -Force
}

# Start background jobs to collect logs
$jobs = @()
foreach ($server in $servers) {
    $user = $server.User
    $hostname = $server.Host
    $octet = $hostname.Split('.')[-1]
    $logFile = "/home/$user/test_ansible_scripts/logs/master_log_$octet.log"
    $tempLog = "C:\temp\logfile_$octet.log"

    $job = Start-Job -ScriptBlock {
        param ($user, $hostname, $logFile, $tempLog)
        ssh $user@$hostname "tail -f $logFile" | Out-File -Append -FilePath $tempLog -Encoding utf8
    } -ArgumentList $user, $hostname, $logFile, $tempLog

    $jobs += $job
}

# Define the merged log file path
$mergedLog = "C:\temp\merged_log.log"

# Continuously merge the collected logs into a single output
Start-Job -ScriptBlock {
    param ($servers, $mergedLog)
    while ($true) {
        $logContents = @()
        foreach ($server in $servers) {
            $octet = $server.Host.Split('.')[-1]
            $tempLog = "C:\temp\logfile_$octet.log"
            if (Test-Path $tempLog) {
                $logContents += Get-Content $tempLog
            }
        }
        $logContents | Set-Content $mergedLog
        Start-Sleep -Seconds 1
    }
} -ArgumentList $servers, $mergedLog

# Construct the Windows Terminal command to display the merged log
### wt.exe new-tab -p "PowerShell" --title "Log Viewer" -- powershell -NoExit -Command "Get-Content C:\temp\merged_log.log -Wait"

$wtCommand = "new-tab  --title 'All_VM_Status' -- powershell  -NoExit -Command `"Get-Content $mergedLog -Wait`""
# Run the command in Windows Terminal
##Start-Process wt.exe -ArgumentList $wtCommand

# Construct the Windows Terminal command for SSH connections
$sshCommand = "new-tab --title '$($servers[0].Host)' ssh $($servers[0].User)@$($servers[0].Host) " +
              "; split-pane -V --title '$($servers[1].Host)' ssh $($servers[1].User)@$($servers[1].Host) " +
              "; new-tab --title '$($servers[2].Host)' ssh $($servers[2].User)@$($servers[2].Host) " +
              "; split-pane -V --title '$($servers[3].Host)' ssh $($servers[3].User)@$($servers[3].Host) " +
              "; split-pane -H --title '$($servers[5].Host)' ssh $($servers[5].User)@$($servers[5].Host) " +
              "; focus-pane -t 0; split-pane -H --title '$($servers[4].Host)' ssh $($servers[4].User)@$($servers[4].Host)" +
			  "; $wtCommand "

# Run the SSH command in Windows Terminal
Start-Process wt.exe -ArgumentList $sshCommand

# Define a cleanup function to stop and remove background jobs
function Cleanup-Jobs {
    Get-Job | Where-Object { $_.State -eq 'Running' } | Stop-Job
    Get-Job | Remove-Job
}

# Trap statement to handle Ctrl+C
trap {
    Cleanup-Jobs
    Write-Host "Script terminated. Background jobs cleaned up."
    exit
}

# Keep the PowerShell script running to prevent the tab from closing
try {
    while ($true) {
        Start-Sleep -Seconds 60
    }
} finally {
    # Cleanup background jobs when the script is terminated
    Cleanup-Jobs
}