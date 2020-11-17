package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"mime/quotedprintable"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var exUnlockCommand = "[QUNLOCK:2]"
var resetCommand = "[QRESET:OK123!]"
var unlockCommand = "[QUNLOCK:<lock_id>]"
var lockCommand = "[QLOCK:<lock_id>]"
var sendSMSCommand = "[QSMS:<phoneno>:<smsbody>:OK123!]"
var cimiCommand = "[OHUB:CIMI:<cimistr>:OK]"
var lockStatusCommand = "[STATUS:<nbrpar>:iiiiiiiiiiii]"
var federal1UnlockCommand = "[FEDERAL1:QUNLOCK:1]"

//Robolink

var setPass = "[QSETPASS:0:123456:OK]"
var returnSetPass = "[RPASS:0:123456:OK]"
var setMasterPass = "[QSETPASS:99:1122334455:OK]"
var errorReturn = "[NACK]"
var clearPass = "[QCLEARPASS:0:OK]"
var successEvent = "[REVENT:1:0:OK]"
var invalidPass = "[REVENT:2:X:OK]"

func sendUnlockCommand() {
	writeHub([]byte(exUnlockCommand))
}

func sendEmail(tcKimlik string, gasType string, lockNumber string) {
	fromemail := "cagricalis@gmail.com"
	password := "f)S3eg-n"
	host := "smtp.gmail.com:587"
	auth := smtp.PlainAuth("", fromemail, password, "smtp.gmail.com")

	header := make(map[string]string)
	toemail := "cagricalis@gmail.com"
	header["From"] = fromemail
	header["To"] = toemail
	header["Subject"] = "AYGAZ Tupmatik Yeni Satis Gerceklesti"

	header["MIME-Version"] = "1.0"
	header["Content-Type"] = fmt.Sprintf("%s; charset=\"utf-8\"", "text/html")
	header["Content-Transfer-Encoding"] = "quoted-printable"
	header["Content-Disposition"] = "inline"

	headermessage := ""
	for key, value := range header {
		headermessage += fmt.Sprintf("%s: %s\r\n", key, value)
	}
	var newMessageFromDevice = fmt.Sprintf("Yeni ürün satışı gerçekleştirildi. TC KIMLIK NO: %s, Satis Yapilan Locker Numarası: %s, Satılan Urun Tipi: %s \r\n", tcKimlik, lockNumber, gasType)

	body := "<h3>" + newMessageFromDevice + "</h3>"
	var bodymessage bytes.Buffer
	temp := quotedprintable.NewWriter(&bodymessage)
	temp.Write([]byte(body))
	temp.Close()

	finalmessage := headermessage + "\r\n" + bodymessage.String()
	status := smtp.SendMail(host, auth, fromemail, []string{toemail}, []byte(finalmessage))
	if status != nil {
		log.Printf("Error from SMTP Server: %s", status)
	}
	log.Print("Email Sent Successfully")
}

func sendFillEmail(fillType string) {
	fromemail := "cagricalis@gmail.com"
	password := "f)S3eg-n"
	host := "smtp.gmail.com:587"
	auth := smtp.PlainAuth("", fromemail, password, "smtp.gmail.com")

	header := make(map[string]string)
	toemail := "cagricalis@gmail.com"
	header["From"] = fromemail
	header["To"] = toemail
	header["Subject"] = "AYGAZ Tupmatik Dolum Gerceklesti"

	header["MIME-Version"] = "1.0"
	header["Content-Type"] = fmt.Sprintf("%s; charset=\"utf-8\"", "text/html")
	header["Content-Transfer-Encoding"] = "quoted-printable"
	header["Content-Disposition"] = "inline"

	headermessage := ""
	for key, value := range header {
		headermessage += fmt.Sprintf("%s: %s\r\n", key, value)
	}
	var newMessageFromDevice = fmt.Sprintf("Yeni dolum kapak sırasına göre şu şekilde gerçekleştirildi: %s -- G: 2KG Dar çemberli küçük tüp, Y: Aygaz Mini, B: Boş \r\n", fillType)

	body := "<h3>" + newMessageFromDevice + "</h3>"
	var bodymessage bytes.Buffer
	temp := quotedprintable.NewWriter(&bodymessage)
	temp.Write([]byte(body))
	temp.Close()

	finalmessage := headermessage + "\r\n" + bodymessage.String()
	status := smtp.SendMail(host, auth, fromemail, []string{toemail}, []byte(finalmessage))
	if status != nil {
		log.Printf("Error from SMTP Server: %s", status)
	}
	log.Print("Email Sent Successfully")
}

var aygazSale = "AYGAZSALE"
var aygazSale1 = "[AYGAZSALE"
var aygazInfo = "[AYGAZINFO"
var saleInfo = "[AYGAZINFO:SALE"
var aygazOpenLockInfo = "AYGAZOPENLOCKINFO"
var statusCommand = "STATUS"
var fillingCommand = "AYGAZINFO"
var aygazSaleAlternative = "[AYGAZINFO:SALE;Y;25687255777]"
var fillingAlternative = "[AYGAZINFO:DOLUM;BBBBBBBBB]"
var hub *net.Conn
var clients []*websocket.Conn

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var hubStatus = "[QR:LOCKS:1111100111XXXXXX:OK]"
var counter = 0

func handleWebsocket() {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			log.Println(err.Error())
			deleteClient(conn)
			return
		}

		log.Println("Client connected!")
		clients = append(clients, conn)

		// :: added

		if err := conn.WriteMessage(1, []byte(hubStatus)); err != nil {
			log.Println("in handlewebsocket  err write message")
			log.Println(err.Error())
		}

		for {
			_, data, err := conn.ReadMessage()

			if err != nil {

				log.Println("in handlewebsocket err")
				log.Println(err.Error())

				// continue
				break
			}

			log.Println("Client:", string(data))
			writeHub(data)

			time.AfterFunc(3*time.Second, sendHubStatus)

			writeClients(string(data))
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "federal.html")
	})
}

func acceptTCP(sw net.Listener) {
	log.Println("acceptTCP()")
	for {
		connection, err := sw.Accept()

		if err != nil {
			log.Println("in acceptTCP err")
			log.Println(err.Error())
			continue
		}

		log.Println("Accepted hub connection!")

		go handleTCP(&connection)
	}
}

func sendHubStatus() {
	writeHub([]byte("[QSTATUS?]"))
	log.Println("->Status")
}
func sendUnlock(s string) {
	writeHub([]byte("[QUNLOCK:" + s + "]"))

}

func sendUnlock1() {

	writeHub([]byte("[QUNLOCK:1]"))
}

var headData = "headData"
var tcKimlik = "1234567890"
var gasType = "gri"
var lockNumber = "5"
var fillType = "GGYYGGGYY"

func isset(arr []string, index int) bool {
	return (len(arr) > index)
}
func replaceAtIndex(in string, r rune, i int) string {
	out := []rune(in)
	out[i] = r
	return string(out)
}

//Robolink
var revent = "[RPASS"
var returnLockNumber = "999"
var lockPassword = "123456"

func handleTCP(c *net.Conn) {
	log.Println("HandleTCP()")
	log.Println("Hub connection request:", (*c).RemoteAddr().String())
	hub = c

	for {
		netData, err := bufio.NewReader(*c).ReadString('\n')
		netData = strings.TrimSpace(string(netData))
		log.Println("netDATA: ", netData)

		s := strings.Split(netData, ":")
		s1, s2, s3 := s[0], s[1], s[2]
		log.Println("s1s2s3s4:", s1, s2, s3)

		if s1 == revent {
			returnLockNumber = s2
			lockPassword = s3
			log.Println("revent splitted:")
			log.Println("s2: ", s2)
			log.Println("s3: ", s3)

		}

		if err != nil {
			log.Println("Hub disconnected:", err)
			hub = nil
			break
		}
		log.Println(string(netData))

		if matches, err := regexp.Match(`^\[STATUS:24:\d{24}]$`, []byte(netData)); err != nil {
			log.Println("NOT MATCHES!")
			log.Println(err.Error())

		} else if matches {
			log.Println("New hub status received:", netData)
			hubStatus = netData
		} else {
			log.Println("Hub:", netData)
		}

		writeClients(netData)
	}

	if err := (*c).Close(); err != nil {
		log.Println("close func!!")
		log.Println(err.Error())
	}
}

// func handleTCP(c *net.Conn) {
// 	log.Println("HandleTCP()")
// 	log.Println("Hub connection request:", (*c).RemoteAddr().String())
// 	hub = c

// 	for {
// 		netData, err := bufio.NewReader(*c).ReadString('\n')
// 		netData = strings.TrimSpace(string(netData))
// 		log.Println("netDATA: ", netData)

// 		split1 := strings.Split(netData, ":")
// 		log.Println("Split1:", split1)
// 		splitted := split1[0]
// 		log.Println(splitted)
// 		if splitted == aygazSale1 {
// 			log.Println("Splitted:", splitted)
// 			if isset(split1, 3) && split1[3] != "" {

// 				splitted2 := split1[3]
// 				log.Println(splitted2)

// 				headData = strings.Trim(splitted, "[")
// 				log.Println("HEADDATA: ", headData)
// 				if aygazSale == headData {
// 					log.Println("AYGAZSALE READED")
// 					lockNumber = split1[1]

// 					log.Printf("%s Nolu dolaptan satis", split1[1])
// 					if split1[2] == "G" {
// 						log.Println("GRI SATILDI")
// 						gasType = "2KG Dar Çemberli Küçük Tüp"
// 					} else if split1[2] == "Y" {
// 						log.Println("YESIL SATILDI")
// 						gasType = "Aygaz Mini"
// 					}
// 					tcKimlik = strings.Trim(splitted2, "]")
// 					log.Printf("TC KIMLIK NO: %s\n", tcKimlik)
// 					sendEmail(tcKimlik, gasType, lockNumber)

// 				}
// 			}

// 		} else if aygazInfo == splitted {
// 			log.Println("AYGAZINFO READED")

// 			log.Println("split[1] ", split1[1])
// 			split1[1] = strings.Trim(split1[1], "]")
// 			fillType = split1[1]

// 			sendFillEmail(fillType)

// 			// f, err := os.OpenFile("fillingdb.txt", os.O_RDWR, 0600)
// 			// if err != nil {
// 			// 	panic(err)
// 			// }

// 			// defer f.Close()
// 			// if _, err = f.WriteString(""); err != nil {
// 			// 	panic(err)
// 			// }
// 			// if _, err = f.WriteString(split1[1]); err != nil {
// 			// 	panic(err)
// 			// }

// 		} else if aygazOpenLockInfo == headData {
// 			log.Println("AYGAZOPENLOCKINFO READED")
// 		} else if statusCommand == headData {
// 			log.Println("STATUS")
// 		} else if fillingCommand == headData {
// 			log.Println("FILLING")
// 			log.Println("NETDATA: ", netData)
// 		}

// 		if err != nil {
// 			log.Println("Hub disconnected:", err)
// 			hub = nil
// 			break
// 		}
// 		log.Println(string(netData))

// 		if matches, err := regexp.Match(`^\[STATUS:12:\d{12}]$`, []byte(netData)); err != nil {
// 			log.Println("NOT MATCHES!")
// 			log.Println(err.Error())

// 		} else if matches {
// 			log.Println("New hub status received:", netData)
// 			hubStatus = netData
// 		} else {
// 			log.Println("Hub:", netData)
// 		}

// 		writeClients(netData)
// 	}

// 	if err := (*c).Close(); err != nil {
// 		log.Println("close func!!")
// 		log.Println(err.Error())
// 	}
// }

func writeHub(data []byte) {
	if hub != nil {
		if _, err := (*hub).Write(data); err != nil {
			log.Println("writehub! func")
			log.Println(err.Error())
		}
	}
}

func writeClients(data string) {
	for i := 0; i < len(clients); i++ {
		if err := clients[i].WriteMessage(1, []byte(string(data))); err != nil {

			log.Println("write clients func!!")
			log.Println(err.Error())
		}
	}
}

func main() {

	log.Println("Started listening TCP on port: 9011")
	log.Println("Started listening Websocket on port: 9012")

	// Listen TCP Clients
	sw, err := net.Listen("tcp4", ":9013")

	if err != nil {
		log.Println("in main err sw")
		log.Println(err.Error())
		return
	}

	go acceptTCP(sw)

	// Listen Websocket Clients
	handleWebsocket()

	if err := http.ListenAndServe(":9014", nil); err != nil {
		log.Println("in main listen and serve err")
		log.Println(err.Error())
		return
	}

}

func deleteClient(c *websocket.Conn) {
	for i := 0; i < len(clients); i++ {
		if clients[i] == c {
			log.Println("in main delete client")
			clients[i] = clients[len(clients)-1]
			clients[len(clients)-1] = nil
			clients = clients[:len(clients)-1]
			return
		}
	}
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// func createFile() {
// 	// check if file exists
// 	var _, err = os.Stat(path)

// 	// create file if not exists
// 	if os.IsNotExist(err) {
// 		var file, err = os.Create(path)
// 		if isError(err) {
// 			return
// 		}
// 		defer file.Close()
// 	}

// 	fmt.Println("File Created Successfully", path)
// }

// func writeFile() {
// 	// Open file using READ & WRITE permission.
// 	var file, err = os.OpenFile(path, os.O_RDWR, 0644)
// 	if isError(err) {
// 		return
// 	}
// 	defer file.Close()

// 	// Write some text line-by-line to file.
// 	_, err = file.WriteString("Hello \n")
// 	if isError(err) {
// 		return
// 	}
// 	_, err = file.WriteString("World \n")
// 	if isError(err) {
// 		return
// 	}

// 	// Save file changes.
// 	err = file.Sync()
// 	if isError(err) {
// 		return
// 	}

// 	fmt.Println("File Updated Successfully.")
// }

// func readFile() {
// 	// Open file for reading.
// 	var file, err = os.OpenFile(path, os.O_RDWR, 0644)
// 	if isError(err) {
// 		return
// 	}
// 	defer file.Close()

// 	// Read file, line by line
// 	var text = make([]byte, 1024)
// 	for {
// 		_, err = file.Read(text)

// 		// Break if finally arrived at end of file
// 		if err == io.EOF {
// 			break
// 		}

// 		// Break if error occured
// 		if err != nil && err != io.EOF {
// 			isError(err)
// 			break
// 		}
// 	}

// 	fmt.Println("Reading from file.")
// 	fmt.Println(string(text))
// }

// func deleteFile() {
// 	// delete file
// 	var err = os.Remove(path)
// 	if isError(err) {
// 		return
// 	}

// 	fmt.Println("File Deleted")
// }

func isError(err error) bool {
	if err != nil {
		fmt.Println(err.Error())
	}

	return (err != nil)
}
