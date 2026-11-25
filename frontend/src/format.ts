export function checkIntervalLabel(seconds: number): string {
  if (seconds % 3600 === 0) {
    const hours = seconds / 3600
    return hours === 1 ? 'Every hour' : `Every ${hours} hours`
  }
  if (seconds % 60 === 0) {
    const minutes = seconds / 60
    return minutes === 1 ? 'Every minute' : `Every ${minutes} minutes`
  }
  return seconds === 1 ? 'Every second' : `Every ${seconds} seconds`
}
