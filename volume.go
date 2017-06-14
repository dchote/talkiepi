package talkiepi

import (
  "fmt"
  "os"
  "os/exec"
  "strconv"

)

func(b *Talkiepi) IncreaseVolume(increment int) {
  fmt.Printf("Volume increased by %v\n", increment)
  changeVolume(strconv.Itoa(increment) + "%+")
}

func(b *Talkiepi) DecreaseVolume(increment int) {
  fmt.Printf("Volume decreased by %v\n", increment)
  changeVolume(strconv.Itoa(increment) + "%-")
}

//Volume changes are handled by adding or subtracting an increment (e.g. "amixer sset Master 10%+")
//amixer will not allow changes beyond 0% or 100%, so we aren't doing any limiting ourselves
func changeVolume(changeString string) {
  var err error

  volCommand := "amixer"
  volCommandArgs := []string{"sset","Master", changeString}

  fmt.Printf("Running: amixer sset Master %v\n", changeString)

  if  err = exec.Command(volCommand, volCommandArgs...).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error changing volume: ", err)
		os.Exit(1)
	}
}
