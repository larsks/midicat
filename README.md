# midicat

Download the binaries for Linux and Windows [here](https://github.com/gomidi/midicat/releases/download/v0.2.0/midicat-binaries.zip).

When using windows, run the commands inside `cmd.exe`.

## Usage / Examples

- get the list of MIDI in ports

    midicat ins
    
- log the input from MIDI in port 11

    midicat in -i=11 | midicat log

- pass the MIDI data from in port 11 to out port 12 while logging it to stderr

    midicat in -i=11 | midicat log | midicat out -i=12
    
- get help

    midicat help
