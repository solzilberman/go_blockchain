package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	//"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

type Block struct{
	Index 	  int
	Timestamp string
	BPM       int
	Hash 	  string
	PrevHash  string
}

// each block should have prev hash == prevBlock hash 
var Blockchain []Block

// function to calc hash for each block
func calcHash(block Block) string {
	record := string(block.Index) + block.Timestamp + string(block.BPM) + block.PrevHash
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

// function to gen new block
func generateBlock(oldBlock Block, BPM int) (Block, error){
	var newBlock Block

	t := time.Now()

	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.BPM = BPM
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Hash = calcHash(newBlock)

	return newBlock, nil
}

// make sure block being added is valid and hash is good
func validateBlock(oldBlock Block, newBlock Block) bool{
	if(oldBlock.Index + 1 != newBlock.Index){
		return false
	}
	if(newBlock.PrevHash != oldBlock.Hash){
		return false
	}
	if(calcHash(newBlock) != newBlock.Hash){
		return false
	}

	return true
}

// if two nodes return chain after validation --> pick longer one
func correctChain(newBlocks []Block){
	if(len(newBlocks) > len(Blockchain)){
		Blockchain = newBlocks
	}
}

// init basic web server so we can interact with blockchain :)
func run() error {
	mux := makeMuxRouter()
	log.Println("Listening on 8080")
	s := &http.Server{
		Addr:		":" + "8080",
		Handler:	mux,
		ReadTimeout:	10 * time.Second,
		WriteTimeout:	10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	
	if err := s.ListenAndServe(); err != nil {
		return err
	}
	
	return nil
}

func makeMuxRouter() http.Handler {
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
	muxRouter.HandleFunc("/", handleWriteBlock).Methods("POST")
	return muxRouter
}

func handleGetBlockchain(w http.ResponseWriter, r *http.Request){
	bytes, err := json.MarshalIndent(Blockchain, "", "	")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, string(bytes))
}

type Message struct {
	BPM int
}

func handleWriteBlock(w http.ResponseWriter, r *http.Request){
	var m Message
	decoder := json.NewDecoder(r.Body)
	log.Println(decoder)	
	if err := decoder.Decode(&m); err != nil{
		respondWithJSON(w, r, http.StatusBadRequest, r.Body)
		return
	}

	defer r.Body.Close()
	
	newBlock, err := generateBlock(Blockchain[len(Blockchain)-1], m.BPM)
	log.Println(newBlock)
	if err != nil {
		respondWithJSON(w, r,  http.StatusInternalServerError, m)
		return
	}

	if validateBlock(newBlock, Blockchain[len(Blockchain)-1]){
		newBlockchain := append(Blockchain, newBlock)
		correctChain(newBlockchain)
		spew.Dump(Blockchain)
	}

	respondWithJSON(w, r, http.StatusCreated, newBlock)
}

func respondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{}){
	response, err := json.MarshalIndent(payload, "", "	")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("HTTP 500: Internal Server Error"))
		return
	}
	w.WriteHeader(code)
	w.Write(response)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		t := time.Now()
		genesisBlock := Block{0, t.String(), 0, "", ""}
		spew.Dump(genesisBlock)
		Blockchain = append(Blockchain, genesisBlock)
	}()
	log.Fatal(run())
}


