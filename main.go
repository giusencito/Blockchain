package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type MessageType int32

const (
	NEWHOST          MessageType = 0
	ADDHOST          MessageType = 1
	ADDBLOCK         MessageType = 2
	NEWBLOCK         MessageType = 3
	SETBLOCKS        MessageType = 4
	NEWBLOCKMESSAGE  MessageType = 5
	SETBLOCKSMESSAGE MessageType = 6
	ADDBLOCKMESSAGE  MessageType = 7
	NEWBLOCKMEDICAL  MessageType = 8
	SETBLOCKSMEDICAL MessageType = 9
	ADDBLOCKMEDICAL  MessageType = 10
	PROTOCOL                     = "tcp"
	NEWMR                        = 1
	LISTMR                       = 2
	LISTHOSTS                    = 3
	NEWMETS                      = 4
	LISTMER                      = 5
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

func BroadcastBlockMedical(newBlock BlockMedical) {
	for _, host := range HOSTS {
		data, _ := json.Marshal(newBlock)
		requestBroadcast := RequestBody{
			Message:     string(data),
			MessageType: ADDBLOCKMEDICAL,
		}
		broadcastMessage, _ := json.Marshal(requestBroadcast)
		SendMessage(host, string(broadcastMessage))
	}
}

func BroadcastBlockMessage(wg *sync.WaitGroup, newBlock BlockMessage) {
	defer wg.Done()
	for _, host := range HOSTS {
		data, _ := json.Marshal(newBlock)
		requestBroadcast := RequestBody{
			Message:     string(data),
			MessageType: ADDBLOCKMESSAGE,
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
		} else if request.MessageType == SETBLOCKSMESSAGE {
			_ = json.Unmarshal([]byte(request.Message), &localBlockChainMessage.Chain)
			updatedBlocks <- 0
		} else if request.MessageType == SETBLOCKSMEDICAL {
			_ = json.Unmarshal([]byte(request.Message), &localBlockChainMedical.Chain)
			updatedBlocks <- 0
		} else if request.MessageType == ADDBLOCK {
			block := Block{}
			src := []byte(request.Message)
			json.Unmarshal(src, &block)
			localBlockChain.Chain = append(localBlockChain.Chain, block)
		} else if request.MessageType == ADDBLOCKMESSAGE {
			block := BlockMessage{}
			src := []byte(request.Message)
			json.Unmarshal(src, &block)
			localBlockChainMessage.Chain = append(localBlockChainMessage.Chain, block)
		} else if request.MessageType == ADDBLOCKMEDICAL {
			block := BlockMedical{}
			src := []byte(request.Message)
			json.Unmarshal(src, &block)
			localBlockChainMedical.Chain = append(localBlockChainMedical.Chain, block)
		} else if request.MessageType == NEWBLOCKMESSAGE {
			blocksMessage, _ := json.Marshal(localBlockChainMessage.Chain)
			setBlocksRequest := RequestBody{
				Message:     string(blocksMessage),
				MessageType: SETBLOCKSMESSAGE,
			}
			setBlocksMessage, _ := json.Marshal(setBlocksRequest)
			SendMessage(request.Message, string(setBlocksMessage))
		} else if request.MessageType == NEWBLOCKMEDICAL {
			blocksMessage, _ := json.Marshal(localBlockChainMedical.Chain)
			setBlocksRequest := RequestBody{
				Message:     string(blocksMessage),
				MessageType: SETBLOCKSMEDICAL,
			}
			setBlocksMessage, _ := json.Marshal(setBlocksRequest)
			SendMessage(request.Message, string(setBlocksMessage))
		}

	}
	end <- 0
}

/******************BLOCKCHAIN**********************/

type Message struct {
	MessagetoSend string
	ToHost        string
	FromHost      string
}

type Login struct {
	Username string
	Password string
}

type PermissionLevel int32
type ConsensusLevel int32

const (
	Read        PermissionLevel = 0
	Write       PermissionLevel = 1
	ReadOrWrite PermissionLevel = 2
	All         ConsensusLevel  = 0
	ONE         ConsensusLevel  = 1
	MAJORITY    ConsensusLevel  = 2
	OWNER       ConsensusLevel  = 3
)

type ThridEntity struct {
	Username  string
	Password  string
	Email     string
	readwrite int
}
type DataKeeper struct {
	Username string
	Password string
	FullName string
	Email    string
}
type Record struct {
	datakeepers    []DataKeeper
	id             int
	ConsensusLevel ConsensusLevel
	MedicalRecord  MedicalRecord
}
type Policy struct {
	entity ThridEntity
	record Record
	id     int
	level  PermissionLevel
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

type BlockMedical struct {
	Index        int
	Timestamp    time.Time
	Data         MedicalRecord
	PreviousHash string
	Hash         string
}

type BlockMessage struct {
	Index        int
	Timestamp    time.Time
	Data         Message
	PreviousHash string
	Hash         string
}

func (block *Block) CalculateHash() string {
	src := fmt.Sprintf("%d-%s-%s", block.Index, block.Timestamp.String(), block.Data)
	return base64.StdEncoding.EncodeToString([]byte(src))
}

func (block *BlockMessage) CalculateHashMessage() string {
	src := fmt.Sprintf("%d-%s-%s", block.Index, block.Timestamp.String(), block.Data)
	return base64.StdEncoding.EncodeToString([]byte(src))
}

func (block *BlockMedical) CalculateHashMedical() string {
	src := fmt.Sprintf("%d-%s-%s", block.Index, block.Timestamp.String(), block.Data)
	return base64.StdEncoding.EncodeToString([]byte(src))
}

type BlockChain struct {
	Chain []Block
}

type BlockChainMessages struct {
	Chain []BlockMessage
}

type BlockChainMedical struct {
	Chain []BlockMedical
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

func (blockChain *BlockChainMessages) CreateGenesisMessageBlock() BlockMessage {
	block := BlockMessage{
		Index:        0,
		Timestamp:    time.Now(),
		Data:         Message{},
		PreviousHash: "0",
	}
	block.Hash = block.CalculateHashMessage()
	return block
}

func (blockChain *BlockChainMedical) CreateGenesisMedicalBlock() BlockMedical {
	block := BlockMedical{
		Index:        0,
		Timestamp:    time.Now(),
		Data:         MedicalRecord{},
		PreviousHash: "0",
	}
	block.Hash = block.CalculateHashMedical()
	return block
}

func (blockChain *BlockChain) GetLatesBlock() Block {
	n := len(blockChain.Chain)
	return blockChain.Chain[n-1]
}

func (blockChain *BlockChainMessages) GetLatesMessageBlock() BlockMessage {
	n := len(blockChain.Chain)
	return blockChain.Chain[n-1]
}

func (blockChain *BlockChainMedical) GetLatesMedicalBlock() BlockMedical {
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

func (blockChain *BlockChainMessages) AddMessageBlock(wg *sync.WaitGroup, block BlockMessage) {
	defer wg.Done()
	block.Timestamp = time.Now()
	block.Index = blockChain.GetLatesMessageBlock().Index + 1
	block.PreviousHash = blockChain.GetLatesMessageBlock().Hash
	block.Hash = block.CalculateHashMessage()
	blockChain.Chain = append(blockChain.Chain, block)
}

func (blockChain *BlockChainMedical) AddMedicalBlock(block BlockMedical) {
	block.Timestamp = time.Now()
	block.Index = blockChain.GetLatesMedicalBlock().Index + 1
	block.PreviousHash = blockChain.GetLatesMedicalBlock().Hash
	block.Hash = block.CalculateHashMedical()
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

func (blockChain *BlockChainMessages) IsMessagesChainValid() bool {
	n := len(blockChain.Chain)
	for i := 1; i < n; i++ {
		currentBlock := blockChain.Chain[i]
		previousBlock := blockChain.Chain[i-1]
		if currentBlock.Hash != currentBlock.CalculateHashMessage() {
			return false
		}
		if currentBlock.PreviousHash != previousBlock.Hash {
			return false
		}
	}
	return true
}

func (blockChain *BlockChainMedical) IsMedicalChainValid() bool {
	n := len(blockChain.Chain)
	for i := 1; i < n; i++ {
		currentBlock := blockChain.Chain[i]
		previousBlock := blockChain.Chain[i-1]
		if currentBlock.Hash != currentBlock.CalculateHashMedical() {
			return false
		}
		if currentBlock.PreviousHash != previousBlock.Hash {
			return false
		}
	}
	return true
}

func CreateBlockChain() BlockChain {
	bc := BlockChain{}
	genesisBlock := bc.CreateGenesisBlock()

	bc.Chain = append(bc.Chain, genesisBlock)
	return bc
}

func CreateMessageBlockChain() BlockChainMessages {
	bc := BlockChainMessages{}
	genesisBlock := bc.CreateGenesisMessageBlock()

	bc.Chain = append(bc.Chain, genesisBlock)
	return bc
}

func CreateMedicalBlockChain() BlockChainMedical {
	bc := BlockChainMedical{}
	genesisBlock := bc.CreateGenesisMedicalBlock()

	bc.Chain = append(bc.Chain, genesisBlock)
	return bc
}

var localBlockChain BlockChain
var localBlockChainMessage BlockChainMessages
var localBlockChainMedical BlockChainMedical
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

func PrintMedicalRecordstwo() {
	blocks := localBlockChainMedical.Chain[1:]
	for index, block := range blocks {
		medicalRecord := block.Data
		fmt.Printf("- - - Medical Record No. %d - - - \n", index+1)
		fmt.Printf("\tName: %s\n", medicalRecord.Name)
		fmt.Printf("\tYear: %s\n", medicalRecord.Year)
		fmt.Printf("\tHospital: %s\n", medicalRecord.Hospital)
		fmt.Printf("\tDoctor: %s\n", medicalRecord.Doctor)
		fmt.Printf("\tDiagnostic: %s\n", medicalRecord.Diagnostic)
		fmt.Printf("\tMedication: %s\n", medicalRecord.Medication)
		fmt.Printf("\tProcedure: %s\n", medicalRecord.Procedure)
	}
}

func PrintMessagesRecords() {
	blocks := localBlockChainMessage.Chain[1:]
	for index, block := range blocks {
		messagetosend := block.Data
		fmt.Printf("- - - Message Record No. %d - - - \n", index+1)
		fmt.Printf("\tMessage: %s\n", messagetosend.MessagetoSend)
		fmt.Printf("\tFrom: %s\n", messagetosend.FromHost)
		fmt.Printf("\tTo: %s\n", messagetosend.ToHost)
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

func removeall(threads []BlockMessage) []BlockMessage {
	var blocksempty []BlockMessage
	threads = blocksempty
	return threads
}

func main() {
	var dest string
	var portreceiver string
	var threadmessages []BlockMessage
	var numbermessages string
	var numbermessagesint int
	contmessages := 0
	var wg sync.WaitGroup
	exist := false
	end := make(chan int)
	updatedBlocks := make(chan int)
	fmt.Print("Enter your host: ")
	fmt.Scanf("%s\n", &LOCALHOST)
	fmt.Print("Enter destination host(Empty to be the first node): ")
	fmt.Scanf("%s\n", &dest)
	go BCIPServer(end, updatedBlocks)
	localBlockChain = CreateBlockChain()
	localBlockChainMedical = CreateMedicalBlockChain()
	localBlockChainMessage = CreateMessageBlockChain()
	if dest != "" {
		requestBody := &RequestBody{
			Message:     LOCALHOST,
			MessageType: NEWHOST,
		}
		requestMessage, _ := json.Marshal(requestBody)
		SendMessage(dest, string(requestMessage))
		requestBody.MessageType = NEWBLOCK
		requestMessage, _ = json.Marshal(requestBody)
		requestBody.MessageType = NEWBLOCKMESSAGE
		requestMessage, _ = json.Marshal(requestBody)
		SendMessage(dest, string(requestMessage))
		requestBody.MessageType = NEWBLOCKMEDICAL
		requestMessage, _ = json.Marshal(requestBody)
		SendMessage(dest, string(requestMessage))
		<-updatedBlocks
	}
	var action int
	fmt.Println("Bienvenido a E-Salud! 😇")
	in := bufio.NewReader(os.Stdin)
	for {
		ThridEntity := ThridEntity{
			Username: "banco",

			Password: "nueva",

			Email:     "banco@email.com",
			readwrite: 2,
		}
		dataKeeper := DataKeeper{
			Username: "data",

			Password: "nueva",

			Email: "data@email.com",
		}
		fmt.Print("Welcome thrid entity \n")
		fmt.Print("1. Add medical reports\n2. see medical reports\n3. List Hosts\n4. New Message to Send\n5. List Messages Records\n")
		fmt.Print("😌 Enter action(1|2|3|4|5):")
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
			newBlockMedical := BlockMedical{
				Data: medicalRecord,
			}
			localBlockChainMedical.AddMedicalBlock(newBlockMedical)
			BroadcastBlockMedical(newBlockMedical)
			record := Record{}
			record.id = len(records) + 1
			record.MedicalRecord = medicalRecord
			list := make([]DataKeeper, 0)
			list = append(list, dataKeeper)
			record.datakeepers = list
			record.ConsensusLevel = OWNER
			policy := Policy{}
			policy.id = len(policies) + 1
			policy.entity = ThridEntity
			policy.record = record
			if ThridEntity.readwrite == 0 {
				policy.level = Read
			}
			if ThridEntity.readwrite == 1 {
				policy.level = Write
			}
			if ThridEntity.readwrite == 2 {
				policy.level = ReadOrWrite
			}
			newBlock := Block{
				Data: policy,
			}
			localBlockChain.AddBlock(newBlock)
			BroadcastBlock(newBlock)
			block := localBlockChain.GetLatesBlock()
			fmt.Println(block.Data.level)
			if block.Data.level == Write || block.Data.level == ReadOrWrite {
				medicalrecords = append(medicalrecords, medicalRecord)
				fmt.Println("You have registered successfully! 😀")
			} else {
				fmt.Println("Peticion rechazada")
			}

			time.Sleep(2 * time.Second)
			PrintMedicalRecords()
		} else if action == LISTMR {
			/*var option string
			fmt.Scanf("%d\n", &option)
			intVar, err := strconv.Atoi(option)
			fmt.Println(err)
			medcilrecordassgin := medicalrecords[intVar]
			record := Record{}
			record.id = len(records) + 1
			record.MedicalRecord = medcilrecordassgin
			list := make([]DataKeeper, 0)
			list = append(list, dataKeeper)
			record.datakeepers = list
			record.ConsensusLevel = OWNER
			policy := Policy{}
			policy.id = len(policies) + 1
			policy.entity = ThridEntity
			policy.record = record
			if ThridEntity.readwrite == 0 {
				policy.level = Read
			}
			if ThridEntity.readwrite == 1 {
				policy.level = Write
			}
			if ThridEntity.readwrite == 2 {
				policy.level = ReadOrWrite
			}
			newBlock := Block{
				Data: policy,
			}
			localBlockChain.AddBlock(newBlock)
			BroadcastBlock(newBlock)
			block := localBlockChain.GetLatesBlock()
			fmt.Println(block.Data.level)
			if block.Data.level == Read || block.Data.level == ReadOrWrite {
				fmt.Println("Medical Record! 😀")
				fmt.Printf("\tName: %s\n", medcilrecordassgin.Name)
				fmt.Printf("\tYear: %s\n", medcilrecordassgin.Year)
				fmt.Printf("\tHospital: %s\n", medcilrecordassgin.Hospital)
				fmt.Printf("\tDoctor: %s\n", medcilrecordassgin.Doctor)
				fmt.Printf("\tDiagnostic: %s\n", medcilrecordassgin.Diagnostic)
				fmt.Printf("\tMedication: %s\n", medcilrecordassgin.Medication)
				fmt.Printf("\tProcedure: %s\n", medcilrecordassgin.Procedure)

			} else {
				fmt.Println("Peticion rechazada")
			}*/
			PrintMedicalRecordstwo()
		} else if action == LISTHOSTS {
			PrintHosts()
		} else if action == NEWMETS {
			contmessages = 0
			messagetosend := Message{}
			fmt.Println("- - - Message sender - receiver Network - E-Salud - - -")
			fmt.Printf("Enter the number of messages to send: ")
			fmt.Scanf("%s\n", &numbermessages)
			numbermessagesint, _ = strconv.Atoi(numbermessages)
			for contmessages < numbermessagesint {
				exist = false
				fmt.Print("Enter the message to send: ")
				messagetosend.MessagetoSend, _ = in.ReadString('\n')
				fmt.Print("Enter the Host to send the message: ")
				fmt.Scanf("%s\n", &portreceiver)
				for _, host := range HOSTS {
					if host == portreceiver {
						exist = true
						break
					} else {
						exist = false
					}
				}

				if exist == true {
					messagetosend.ToHost = portreceiver
					messagetosend.FromHost = LOCALHOST
					newBlock := BlockMessage{
						Data: messagetosend,
					}
					threadmessages = append(threadmessages, newBlock)
				} else {
					fmt.Printf("\nNot exist the receiver\n")
					break
				}
				contmessages++
			}

			if len(threadmessages) > 0 {
				for i := 0; i < len(threadmessages); i++ {
					wg.Add(1)
					go localBlockChainMessage.AddMessageBlock(&wg, threadmessages[i])
				}
				wg.Wait()
				for i := 0; i < len(threadmessages); i++ {
					wg.Add(1)
					go BroadcastBlockMessage(&wg, threadmessages[i])
				}
				wg.Wait()

				fmt.Println("\nYou have send All the mesages successfully! 😀")
				time.Sleep(2 * time.Second)
				PrintMessagesRecords()
			}
			threadmessages = removeall(threadmessages)
		} else if action == LISTMER {
			PrintMessagesRecords()
		}
	}
	<-end
}
