package midigen

import (
	"github.com/Oliv95/markov"
	"github.com/algoGuy/EasyMIDI/smf"
	"github.com/algoGuy/EasyMIDI/smfio"
	"io"
	"log"
)

type dataPoint struct {
	dtime      uint32
	status     uint8
	firstBye   byte
	secondByte byte
}

type MidiGen struct {
	state *markov.Graph
}

func addToTrack(track *smf.Track, dtime uint32, status uint8, channel uint8, firstByte uint8, secondByte uint8) error {
	midiEvent, err := smf.NewMIDIEvent(dtime, status, channel, firstByte, secondByte)
	if err != nil {
		return err
	}
	err = track.AddEvent(midiEvent)
	if err != nil {
		return err
	}
	return nil
}

func writeMIDI(writer io.Writer, data []dataPoint) error {
	division, err := smf.NewDivision(960, smf.NOSMTPE)
	if err != nil {
		return err
	}

	midiOut, err := smf.NewSMF(smf.Format0, *division)
	if err != nil {
		return err
	}

	track := &smf.Track{}

	err = midiOut.AddTrack(track)
	if err != nil {
		return err
	}

	for i := 0; i < len(data); i++ {
		dtime := data[i].dtime
		status := data[i].status
		firstByte := data[i].firstBye
		secondByte := data[i].secondByte
		err = addToTrack(track, dtime, status, 0, firstByte, secondByte)
		if err != nil {
			return err
		}
	}

	endOfTrack, err := smf.NewMetaEvent(0, smf.MetaEndOfTrack, []byte{})
	if err != nil {
		return err
	}

	err = track.AddEvent(endOfTrack)
	if err != nil {
		return err
	}

	err = smfio.Write(writer, midiOut)
	if err != nil {
		return err
	}

	return nil
}

func getSMFData(reader io.Reader) ([]dataPoint, error) {
	midi, err := smfio.Read(reader)

	if err != nil {
		return []dataPoint{}, err
	}

	var data []dataPoint
	tracks := midi.GetTracksNum()
	for i := uint16(0); i < tracks; i++ {

		track := midi.GetTrack(i)
		iter := track.GetIterator()

		for iter.MoveNext() {
			value := iter.GetValue()
			// Skip all the meta events, could perhaps use these in the future
			if !smf.CheckMetaStatus(value.GetStatus()) {
				dtime := value.GetDTime()
				status := value.GetStatus()
				bytes := value.GetData()
				if !(len(bytes) < 2) {
					firstByte := bytes[0]
					secondByte := bytes[1]
					data = append(data, dataPoint{dtime, status, firstByte, secondByte})
				}

			}
		}
	}
	return data, nil
}

func populateMarkov(graph *markov.Graph, data []markov.State) {
	for i := 0; i < len(data)-1; i++ {
		from := data[i]
		to := data[i+1]
		markov.AddTransition(graph, from, to)
	}
}

func generate(graph *markov.Graph, start markov.State, iterations int) ([]markov.State, error) {
	result := []markov.State{}
	result = append(result, start)

	current := start
	for i := 0; i < iterations; i++ {
		next, err := markov.Transition(graph, current)
		if err != nil {
			// Return the all the data generated until the error
			return result, err
		}
		result = append(result, *next)
		current = *next
	}
	return result, nil
}

func EmptyGenerator() MidiGen {
	graph := markov.CreateEmptyGraph()
	return MidiGen{
		&graph,
	}
}

// PopulateGraph reads the data from the reader into the graph
func PopulateGraph(generator *MidiGen, reader io.Reader) error {
	data, err := getSMFData(reader)
	if err != nil {
		return err
	}
	// Stop att the second to last since the last element does not have a "to" state
	for i := 0; i < len(data)-1; i++ {
		from := markov.State{Data: data[i]}
		to := markov.State{Data: data[i+1]}
		markov.AddTransition(generator.state, from, to)
	}
	return nil

}

// GenerateMidi generates a midi file
// The generated midi file is written to the out writer
func GenerateMidi(generator *MidiGen, out io.Writer, iterations int) error {
	graph := generator.state
	start, err := markov.RandomState(graph)
	if err != nil {
		return err
	}
	result, err := generate(graph, start, iterations)
	if err != nil {
		// TODO add better log message
		log.Println("non fatal error during midigeneration")
	}
	// Map slice to correct type (Markov.State -> dataPoint)
	dataPoints := []dataPoint{}
	for _, state := range result {
		dataPoint, ok := state.Data.(dataPoint)
		if ok {
			dataPoints = append(dataPoints, dataPoint)
		}
	}
	err = writeMIDI(out, dataPoints)
	if err != nil {
		return err
	}
	return nil
}
