Start-Sleep -Seconds 12
$runs = (Invoke-RestMethod -Uri 'https://api.github.com/repos/amrubio27/kdoctor-mobi-ai-fix/actions/runs?per_page=5').workflow_runs
$relRun = $runs | Where-Object { $_.name -eq 'Release Binaries' } | Select-Object -First 1
Write-Output "Tracking Release Binaries ID: $($relRun.id), Status: $($relRun.status)"
while ($relRun.status -ne 'completed') {
    Start-Sleep -Seconds 6
    $relRun = Invoke-RestMethod -Uri "https://api.github.com/repos/amrubio27/kdoctor-mobi-ai-fix/actions/runs/$($relRun.id)"
    Write-Output "Status: $($relRun.status) / $($relRun.conclusion)"
}
$jobs = Invoke-RestMethod -Uri "https://api.github.com/repos/amrubio27/kdoctor-mobi-ai-fix/actions/runs/$($relRun.id)/jobs"
$jobs.jobs[0].steps | Format-Table name, status, conclusion
