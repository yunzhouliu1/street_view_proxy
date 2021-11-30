package config

const (
	Port             = 8080
	MinX             = 13
	MinY             = 6
	MaxX             = 19
	MaxY             = 10
	Zoom             = 5
	InputSizeX       = 512
	InputSizeY       = InputSizeX
	OutputSizeX      = InputSizeX * (MaxX - MinX)
	OutputSizeY      = InputSizeY * (MaxY - MinY)
	WorkerMultiplier = 10
	SaveImages       = true
	NumRetries       = 10
)
