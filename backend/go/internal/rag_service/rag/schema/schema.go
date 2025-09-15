package schema

const (
	// MetadataKeyImage is the key for image content.
	// The value should be a slice of byte slices ([][]byte), where each inner slice is the raw binary data of an image.
	MetadataKeyImage = "image"
	// MetadataKeyChart is the key for chart or table data.
	// The value can be structured data, like a CSV string or a slice of slices representing the table.
	MetadataKeyChart = "chart"
	// MetadataKeyVideo is the key for video content.
	MetadataKeyVideo = "video"
	// MetadataKeyAudio is the key for audio content.
	MetadataKeyAudio = "audio"
	// MetadataKeyFileName is the key for the source file name.
	MetadataKeyFileName = "file_name"
	// MetadataKeyPageLabel is the key for the page number or label from the source document.
	MetadataKeyPageLabel = "page_label"
)

// Document is the central data structure representing a piece of text and its associated data.
// It is the primary data carrier throughout the RAG pipeline.
type Document struct {
	// ID is the unique identifier for this document chunk.
	ID string

	// Text is the string content of the document chunk.
	Text string

	// Embedding is the vector representation of the text.
	Embedding []float32

	// Metadata holds arbitrary data about the document.
	// It is used to store information like file_name, page_label, type (text, table), etc.
	Metadata map[string]interface{}
}
