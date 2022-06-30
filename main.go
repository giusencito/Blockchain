package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
	"strconv"
)

type MessageType int32

const (
	NEWHOST   MessageType = 0
	ADDHOST   MessageType = 1
	ADDBLOCK  MessageType = 2
	NEWBLOCK  MessageType = 3
	SETBLOCKS MessageType = 4
	PROTOCOL              = "tcp"
	NEWMR                 = 1
	LISTMR                = 2
	LISTHOSTS             = 3
)

/******************BCIP**********************/
var HOSTS []string
var LOCALHOST string

type RequestBody struct {
	Message     string
	MessageType MessageType
}

func GetMessage(conn net.Conn) string {
	reader := bufio.NewReader(conn)
	data, _ := reader.ReadString('\n')
	return strings.TrimSpace(data)
}

func SendMessage(toHost string, message string) {
	conn, _ := net.Dial(PROTOCOL, toHost)
	defer conn.Close()
	fmt.Fprintln(conn, message)
}

func SendMessageWithReply(toHost string, message string) string {
	conn, _ := net.Dial(PROTOCOL, toHost)
	defer conn.Close()
	fmt.Fprintln(conn, message)
	return GetMessage(conn)
}

func RemoveHost(index int, hosts []string) []string {
	n := len(hosts)
	hosts[index] = hosts[n-1]
	hosts[n-1] = ""
	return hosts[:n-1]
}

func RemoveHostByValue(ip string, hosts []string) []string {
	for index, host := range hosts {
		if host == ip {
			return RemoveHost(index, hosts)
		}
	}
	return hosts
}

func Broadcast(newHost string) {
	for _, host := range HOSTS {
		data := append(HOSTS, newHost, LOCALHOST)
		data = RemoveHostByValue(host, data)
		requestBroadcast := RequestBody{
			Message:     strings.Join(data, ","),
			MessageType: ADDHOST,
		}
		broadcastMessage, _ := json.Marshal(requestBroadcast)
		SendMessage(host, string(broadcastMessage))
	}
}

func BroadcastBlock(newBlock Block) {
	for _, host := range HOSTS {
		data, _ := json.Marshal(newBlock)
		requestBroadcast := RequestBody{
			Message:     string(data),
			MessageType: ADDBLOCK,
		}
		broadcastMessage, _ := json.Marshal(requestBroadcast)
		SendMessage(host, string(broadcastMessage))
	}
}

func BCIPServer(end chan<- int, updatedBlocks chan<- int) {
	ln, _ := net.Listen(PROTOCOL, LOCALHOST)
	defer ln.Close()
	for {
		conn, _ := ln.Accept()
		defer conn.Close()
		request := RequestBody{}
		data := GetMessage(conn)
		_ = json.Unmarshal([]byte(data), &request)
		if request.MessageType == NEWHOST {
			message := strings.Join(append(HOSTS, LOCALHOST), ",")
			requestClient := RequestBody{
				Message:     message,
				MessageType: ADDHOST,
			}
			clientMessage, _ := json.Marshal(requestClient)
			SendMessage(request.Message, string(clientMessage))
			Broadcast(request.Message)
			HOSTS = append(HOSTS, request.Message)
		} else if request.MessageType == ADDHOST {
			HOSTS = strings.Split(request.Message, ",")
		} else if request.MessageType == NEWBLOCK {
			blocksMessage, _ := json.Marshal(localBlockChain.Chain)
			setBlocksRequest := RequestBody{
				Message:     string(blocksMessage),
				MessageType: SETBLOCKS,
			}
			setBlocksMessage, _ := json.Marshal(setBlocksRequest)
			SendMessage(request.Message, string(setBlocksMessage))
		} else if request.MessageType == SETBLOCKS {
			_ = json.Unmarshal([]byte(request.Message), &localBlockChain.Chain)
			updatedBlocks <- 0
		} else if request.MessageType == ADDBLOCK {
			block := Block{}
			src := []byte(request.Message)
			json.Unmarshal(src, &block)
			localBlockChain.Chain = append(localBlockChain.Chain, block)
		}
	}
	end <- 0
}

/******************BLOCKCHAIN**********************/

type Login struct {
	Username       string
	Password       string
}

type PermissionLevel int32
type ConsensusLevel int32
const (

Read  PermissionLevel =0
Write  PermissionLevel =1
ReadOrWrite  PermissionLevel =2
All      ConsensusLevel=0
ONE      ConsensusLevel=1
MAJORITY      ConsensusLevel=2
OWNER      ConsensusLevel=3
)

type ThridEntity struct{
	Username       string
	Password       string
	Email          string
	readwrite      int

}
type DataKeeper struct{
	Username       string
	Password       string
	FullName       string
	Email          string
}
type Record struct{
    datakeepers  []DataKeeper
	id           int
    ConsensusLevel ConsensusLevel
	MedicalRecord  MedicalRecord
}
type Policy struct{
  entity ThridEntity
  record   Record
  id       int
  level    PermissionLevel

}


type MedicalRecord struct {
	Name       string
	Year       string
	Hospital   string
	Doctor     string
	Diagnostic string
	Medication string
	Procedure  string
}

type Block struct {
	Index        int
	Timestamp    time.Time
	Data         Policy
	PreviousHash string
	Hash         string
}

func (block *Block) CalculateHash() string {
	src := fmt.Sprintf("%d-%s-%s", block.Index, block.Timestamp.String(), block.Data)
	return base64.StdEncoding.EncodeToString([]byte(src))
}

type BlockChain struct {
	Chain []Block
}

func (blockChain *BlockChain) CreateGenesisBlock() Block {
	block := Block{
		Index:        0,
		Timestamp:    time.Now(),
		Data:         Policy{},
		PreviousHash: "0",
	}
	block.Hash = block.CalculateHash()
	return block
}

func (blockChain *BlockChain) GetLatesBlock() Block {
	n := len(blockChain.Chain)
	return blockChain.Chain[n-1]
}

func (blockChain *BlockChain) AddBlock(block Block) {
	block.Timestamp = time.Now()
	block.Index = blockChain.GetLatesBlock().Index + 1
	block.PreviousHash = blockChain.GetLatesBlock().Hash
	block.Hash = block.CalculateHash()
	blockChain.Chain = append(blockChain.Chain, block)
}

func (blockChain *BlockChain) IsChainValid() bool {
	n := len(blockChain.Chain)
	for i := 1; i < n; i++ {
		currentBlock := blockChain.Chain[i]
		previousBlock := blockChain.Chain[i-1]
		if currentBlock.Hash != currentBlock.CalculateHash() {
			return false
		}
		if currentBlock.PreviousHash != previousBlock.Hash {
			return false
		}
	}
	return true
}

func CreateBlockChain() (BlockChain) {
	bc := BlockChain{}
	genesisBlock := bc.CreateGenesisBlock()
	
	bc.Chain = append(bc.Chain, genesisBlock)
	return bc
}

var localBlockChain BlockChain
var policies []Policy
var records []Record
var medicalrecords []MedicalRecord
/******************MAIN**********************/

func PrintMedicalRecords() {

	for block := range medicalrecords {
		medicalRecord := medicalrecords[block]
	
		fmt.Printf("\tName: %s\n", medicalRecord.Name)
		fmt.Printf("\tYear: %s\n", medicalRecord.Year)
		fmt.Printf("\tHospital: %s\n", medicalRecord.Hospital)
		fmt.Printf("\tDoctor: %s\n", medicalRecord.Doctor)
		fmt.Printf("\tDiagnostic: %s\n", medicalRecord.Diagnostic)
		fmt.Printf("\tMedication: %s\n", medicalRecord.Medication)
		fmt.Printf("\tProcedure: %s\n", medicalRecord.Procedure)
		fmt.Println()
		fmt.Print(len(localBlockChain.Chain))
	}
}

func PrintHosts() {
	fmt.Println("- - - HOSTS - - -")
	const first = 0
	fmt.Printf("\t%s (Your host)\n", LOCALHOST)
	for _, host := range HOSTS {
		fmt.Printf("\t%s\n", host)
	}
}

func main() {
	var dest string
	end := make(chan int)
	updatedBlocks := make(chan int)
	fmt.Print("Enter your host: ")
	fmt.Scanf("%s\n", &LOCALHOST)
	fmt.Print("Enter destination host(Empty to be the first node): ")
	fmt.Scanf("%s\n", &dest)
	go BCIPServer(end, updatedBlocks)
	localBlockChain = CreateBlockChain()
	if dest != "" {
		requestBody := &RequestBody{
			Message:     LOCALHOST,
			MessageType: NEWHOST,
		}
		requestMessage, _ := json.Marshal(requestBody)
		SendMessage(dest, string(requestMessage))
		requestBody.MessageType = NEWBLOCK
		requestMessage, _ = json.Marshal(requestBody)
		SendMessage(dest, string(requestMessage))
		<-updatedBlocks
	}
	var action int
	fmt.Println("Bienvenido a E-Salud! 😇")
	in := bufio.NewReader(os.Stdin)
	for {
		ThridEntity :=ThridEntity{
			Username: "banco",
			
			Password: "nueva",

			Email: "banco@email.com",
			readwrite:     2,

		}
		dataKeeper :=DataKeeper{
			Username: "data",
			
			Password: "nueva",

			Email: "data@email.com",
		}
		fmt.Print("Welcome thrid entity \n")
		fmt.Print("1. Add medical reports\n2. see medical reports\n3. List Hosts\n")
		fmt.Print("😌 Enter action(1|2|3):")
		fmt.Scanf("%d\n", &action)
		if action == NEWMR {
			medicalRecord := MedicalRecord{}
			fmt.Println("- - - Register - - -")
			fmt.Print("Enter name: ")
			medicalRecord.Name, _ = in.ReadString('\n')
			fmt.Print("Enter year: ")
			medicalRecord.Year, _ = in.ReadString('\n')
			fmt.Print("Enter hospital: ")
			medicalRecord.Hospital, _ = in.ReadString('\n')
			fmt.Print("Enter doctor: ")
			medicalRecord.Doctor, _ = in.ReadString('\n')
			fmt.Print("Enter diagnostic: ")
			medicalRecord.Diagnostic, _ = in.ReadString('\n')
			fmt.Print("Enter medication: ")
			medicalRecord.Medication, _ = in.ReadString('\n')
			fmt.Print("Enter procedure: ")
			medicalRecord.Procedure, _ = in.ReadString('\n')
		    record :=Record{}
			record.id=len(records)+1
			record.MedicalRecord=medicalRecord
			list:= make([]DataKeeper,0)
			list = append(list,dataKeeper)
			record.datakeepers=list
			record.ConsensusLevel= OWNER
			policy := Policy{}
			policy.id=len(policies)+1
			policy.entity=ThridEntity
			policy.record=record
			if(ThridEntity.readwrite==0){
				policy.level=Read
			}
			if(ThridEntity.readwrite==1){
				policy.level=Write
			}
			if(ThridEntity.readwrite==2){
				policy.level=ReadOrWrite
			}
			newBlock := Block{
				Data: policy,
			}
			localBlockChain.AddBlock(newBlock)
			BroadcastBlock(newBlock)
			block:=localBlockChain.GetLatesBlock()
			fmt.Println(block.Data.level)
			if(block.Data.level==Write || block.Data.level==ReadOrWrite){
                  medicalrecords=append(medicalrecords,medicalRecord)
				  fmt.Println("You have registered successfully! 😀")
			}else{
				fmt.Println("Peticion rechazada")
			}
			
			time.Sleep(2 * time.Second)
			PrintMedicalRecords()
		} else if action == LISTMR {
			
			var option string
			fmt.Scanf("%d\n", &option)
			intVar, err := strconv.Atoi(option)
			fmt.Println(err)
			medcilrecordassgin:=medicalrecords[intVar]
			record :=Record{}
			record.id=len(records)+1
			record.MedicalRecord=medcilrecordassgin
			list:= make([]DataKeeper,0)
			list = append(list,dataKeeper)
			record.datakeepers=list
			record.ConsensusLevel= OWNER
			policy := Policy{}
			policy.id=len(policies)+1
			policy.entity=ThridEntity
			policy.record=record
			if(ThridEntity.readwrite==0){
				policy.level=Read
			}
			if(ThridEntity.readwrite==1){
				policy.level=Write
			}
			if(ThridEntity.readwrite==2){
				policy.level=ReadOrWrite
			}
			newBlock := Block{
				Data: policy,
			}
			localBlockChain.AddBlock(newBlock)
			BroadcastBlock(newBlock)
			block:=localBlockChain.GetLatesBlock()
			fmt.Println(block.Data.level)
			if(block.Data.level==Read || block.Data.level==ReadOrWrite){
				fmt.Println("Medical Record! 😀")
				fmt.Printf("\tName: %s\n", medcilrecordassgin.Name)
				fmt.Printf("\tYear: %s\n", medcilrecordassgin.Year)
				fmt.Printf("\tHospital: %s\n", medcilrecordassgin.Hospital)
				fmt.Printf("\tDoctor: %s\n", medcilrecordassgin.Doctor)
				fmt.Printf("\tDiagnostic: %s\n", medcilrecordassgin.Diagnostic)
				fmt.Printf("\tMedication: %s\n", medcilrecordassgin.Medication)
				fmt.Printf("\tProcedure: %s\n", medcilrecordassgin.Procedure)
				  
			}else{
				fmt.Println("Peticion rechazada")
			}
		} else if action == LISTHOSTS {
			PrintHosts()
		}
	}
	<-end
}

