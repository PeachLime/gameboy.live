package server

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
//	"encoding/binary"

	"github.com/HFO4/gbc-in-cloud/driver"
	"github.com/HFO4/gbc-in-cloud/gb"
	"github.com/logrusorgru/aurora"
)

// Player Single player model
type Player struct {
	Conn     net.Conn
	Emulator *gb.Core
	ID       string
	Selected int
	GameList *[]GameInfo

	SelectedPlayer   int
	SelectedPlayerID string
}

// Send TELNET options
func (player *Player) InitTelnet() bool {
	// Send telnet options
	_, err := player.Conn.Write([]byte{255, 253, 34})
	_, err = player.Conn.Write([]byte{255, 250, 34, 1, 0, 255, 240})
	// NOT ECHO
	_, err = player.Conn.Write([]byte{0xFF, 0xFB, 0x01})
	if err != nil {
		return false
	}
	return true
}

func (player *Player) Init() bool {

	if player.Emulator == nil {
		Driver := &driver.ASCII{
			Conn: player.Conn,
		}

		core := &gb.Core{
			// Terminal gaming dose not require high FPS,
			// 10 FPS is a decent choice in most situation.
			FPS:           10,
			Clock:         4194304,
			Debug:         false,
			DisplayDriver: Driver,
			Controller:    new(driver.TelnetController),
			DrawSignal:    make(chan bool),
			SpeedMultiple: 0,
			ToggleSound:   false,
		}

		player.Emulator = core

		log.Println("New Player:", player.ID)

	}
	return true

}

// Search Page
func (player *Player) RenderSearchScreen() []byte {
	res := "\033[H"
	res += "Welcome to " + "Search Page" + "\r\n"
	res += fmt.Stringer(aurora.Gray(1-1, " TAB ").BgGray(24-1)).String() + " is to show whole list of GameList" + "\r\n"
	res += fmt.Stringer(aurora.Gray(1-1, " BackSpace ").BgGray(24-1)).String() + " is to clear the current page" + "\r\n"


	return []byte(res)
}

func (player *Player) SearchScreen() int {

	//Clean screen
	_, err := player.Conn.Write([]byte("\033[2J\033[H"))
	if err != nil {
		return -1
	}

	player.Init()



	tmp_gameList_TitleOnly := make([]string,0)
	tmpbuf := make([]byte, 0)
	player_tmpbuf := make([]byte, 0)
	player_inputString := player_tmpbuf

	for index, line := range *player.GameList {
		//fmt.Println(index, line)
		var title = "Title: " + line.Title + " | index: " + strconv.Itoa(index+1)
		tmp_gameList_TitleOnly = append(tmp_gameList_TitleOnly, title)
	}
	
	for {
		var n int
		_, err = player.Conn.Write(player.RenderSearchScreen())
		buf := make([]byte, 512)

		n, err = player.Conn.Read(buf)
		inputKey := buf[:n]
		if err != nil {
			return -1
		}
//testline
		tmpbuf = append(tmpbuf, buf[0])
		inputString := tmpbuf
		if 31 < buf[0] {
			player_tmpbuf = append(player_tmpbuf, buf[0])
			player_inputString = player_tmpbuf
		}

		fmt.Println("tmpbufraw: ", tmpbuf)
		fmt.Println("tmpbuf: ", string(tmpbuf))
		fmt.Println("bufraw: ", buf[0])
		fmt.Println("buf: ", string(buf[0]))
		fmt.Println("inputstring: ", string(inputString))//testline
		player.Conn.Write(player_inputString)




		switch inputKey[len(inputKey)-1] {
		// Enter key pressed => reset bufs
		case 13, 10, 0:
			//fmt.Println(inputKey, strconv.Atoi(string(inputKey)), "\r\n")
			player.Conn.Write([]byte("\033[2J \r\n\n"))

			inputString = inputString[:len(inputString)-1]
	
			fmt.Println("Enter pressed for Search")
			for index, line := range tmp_gameList_TitleOnly {
				fmt.Println("Searched index: ", index)
				var res = strings.Contains(line, string(inputString))
				var searchoutput = strconv.FormatBool(res) + " index: " + strconv.Itoa(index+1) + "\r\n"
				if res == true {
					player.Conn.Write([]byte(searchoutput))
					player.Conn.Write([]byte(line + "\r\n\n"))
				}
			}
				
			tmpbuf = []byte{}
			inputString = []byte{}
			player_tmpbuf = []byte{}
			player_inputString = []byte{}
		//Backspace key pressed		
		case 127:
			fmt.Println("Backspace pressed for clear")
			player.Conn.Write([]byte("\033[2J"))
			player.Conn.Write([]byte("\r\n\n"))
			tmpbuf = []byte{}
			inputString = []byte{}
			player_tmpbuf = []byte{}
			player_inputString = []byte{}
		// ESC key pressed
		case 27:
			player.Conn.Write([]byte("\033[2J"))
			tmpbuf = []byte{}
			inputString = []byte{}
			player_tmpbuf = []byte{}
			player_inputString = []byte{}
			return 0
		//TAB key pressed
		case 9:
			fmt.Println("TAB pressed for show whole list")
			player.Conn.Write([]byte("\033[2J"))
			fmt.Print("shown index: ")
			for index, line := range tmp_gameList_TitleOnly {
				player.Conn.Write([]byte("\r\n" + line))
				fmt.Print(index, " ")
			}
			fmt.Println()
			tmpbuf = []byte{}
			inputString = []byte{}
			player_tmpbuf = []byte{}
			player_inputString = []byte{}

		}

	}

}

// Generate welcome and game selection screen
func (player *Player) RenderWelcomeScreen() []byte {
	res := "\033[H"
	res += "Welcome to " + fmt.Stringer(aurora.Bold(aurora.Green("Gameboy.Live"))).String() + ", you can enjoy GAMEBOY games in your terminal with \"cloud gaming\" experience.\r\n"
	res += "Use " + fmt.Stringer(aurora.Gray(1-1, "Direction keys").BgGray(24-1)).String() + " in your keyboard to select a game, " + fmt.Stringer(aurora.Gray(1-1, " Enter ").BgGray(24-1)).String() + " key to confirm, " + fmt.Stringer(aurora.Gray(1-1, " M ").BgGray(24-1)).String() + " key to enter multi-player mode and select a partner.\r\n"
	res += "Press" + fmt.Stringer(aurora.Gray(1-1, " S ").BgGray(24-1)).String() + "for Search Mode. \r\n"
	res += "\r\n\r\n"

	for k, v := range *player.GameList {
		if player.Selected == k {
			res += "    " + fmt.Stringer(aurora.Gray(1-1, strconv.Itoa(k+1)+".  "+v.Title+"\r\n").BgGray(24-1)).String()
		} else {
			res += "    " + strconv.Itoa(k+1) + ".  " + v.Title + "\r\n"
		}

	}

	res += "\r\n\r\n" + fmt.Stringer(aurora.Yellow("This service is only playable in terminals with ANSI standard and UTF-8 charset support.")).String() + "\r\n"
	res += "Source code of this project is available at: " + fmt.Stringer(aurora.Underline("https://github.com/HFO4/gameboy.live")).String() + " \r\n"
	return []byte(res)
}

/*
	Show the welcome and game select screen, return
	selected game ID.
*/
func (player *Player) Welcome() int {
	//Clean screen
	_, err := player.Conn.Write([]byte("\033[2J\033[H"))
	if err != nil {
		return -1
	}

	player.Init()

	for {
		player.Conn.Write([]byte("Welcome() test line"))

		var n int
		_, err = player.Conn.Write(player.RenderWelcomeScreen())
		buf := make([]byte, 512)
		n, err = player.Conn.Read(buf)
		inputKey := buf[:n]
		if err != nil {
			return -1
		}

		switch inputKey[len(inputKey)-1] {
		// Up key pressed
		case 65:
			if player.Selected == 0 {
				player.Selected = len(*player.GameList) - 1
			} else {
				player.Selected--
			}
		// Down key pressed
		case 66:
			if player.Selected == len(*player.GameList)-1 {
				player.Selected = 0
			} else {
				player.Selected++
			}
		// Enter key pressed
		case 10, 0:
			return player.Selected
		// M key pressed
		case 109:
			player.SelectPlayer()
			_, err = player.Conn.Write([]byte("\033[2J\033[H"))

			// If choose each other, connect their serial driver
			if player.SelectedPlayerID != "" && PlayerList[player.SelectedPlayer].SelectedPlayerID == player.ID {
				PlayerList[player.SelectedPlayer].Emulator.Serial.SetTarget(&player.Emulator.Serial)
				player.Emulator.Serial.SetTarget(&PlayerList[player.SelectedPlayer].Emulator.Serial)
				log.Printf("[Serial] Player %s connect with Player %s", player.SelectedPlayerID, PlayerList[player.SelectedPlayer].SelectedPlayerID)
			}
		// S key pressed => activate Search 
		case 115:
			player.Conn.Write([]byte("testline"))
			search := player.SearchScreen()
			fmt.Println(search)
		
		// Q key pressed => activate logout
		case 113:
			log.Println("User quit")
			player.Emulator.Exit = true
			err := player.Conn.Close()
			if err != nil {
				log.Println("Failed to close connection")
			}
			player.Logout()
			return 0
		}

	}

}

/*
	Render select multiplayer screen
*/
func (player *Player) RenderSelectPlayer() []byte {
	res := "\033[2J\033[H"
	res += "You can play multiplayer game with your friend or strangers. The list below lists players who are currently online. Both of you need to choose each other, so that the connection can be established.\r\n"
	res += "Your player ID: " + fmt.Stringer(aurora.Gray(1-1, player.ID).BgGray(24-1)).String() + "\r\n"
	res += "Player list (Press R to refresh):\r\n\r\n"

	for k, v := range PlayerList {

		if player.SelectedPlayer == k {
			res += "    " + fmt.Stringer(aurora.Gray(1-1, v.ID+"\r\n").BgGray(24-1)).String()
		} else {
			res += "    " + v.ID + "\r\n"
		}

	}
	return []byte(res)
}

/*
	Select multiplayer
*/
func (player *Player) SelectPlayer() int {

	for {
		var n int
		_, err := player.Conn.Write(player.RenderSelectPlayer())
		if err != nil {
			return -1
		}
		buf := make([]byte, 512)
		n, err = player.Conn.Read(buf)
		inputKey := buf[:n]
		if err != nil {
			return -1
		}

		switch inputKey[len(inputKey)-1] {
		// Up key pressed
		case 65:
			if player.SelectedPlayer == 0 {
				player.SelectedPlayer = len(PlayerList) - 1
			} else {
				player.SelectedPlayer--
			}
		// Down key pressed
		case 66:
			if player.SelectedPlayer == len(PlayerList)-1 {
				player.SelectedPlayer = 0
			} else {
				player.SelectedPlayer++
			}
		// Enter key pressed
		case 10, 0:
			// Cannot choose yourself
			if PlayerList[player.SelectedPlayer].ID == player.ID {
				continue
			}

			// Choose none
			if player.SelectedPlayer == 0 {
				player.SelectedPlayerID = ""
				return 0
			}

			player.SelectedPlayerID = PlayerList[player.SelectedPlayer].ID
			return 0
		// R key pressed
		case 114:
			continue
		}

		log.Println(inputKey)
	}
	return 0
}

/*
	Generate the control instruction screen,
	ascii art by Joan Stark.
*/

func (player *Player) Instruction() int {
	ret := "Here's the key instruction, press " + fmt.Stringer(aurora.Gray(1-1, "Enter").BgGray(24-1)).String() + " key to enter the game, " + fmt.Stringer(aurora.Gray(1-1, " Q ").BgGray(24-1)).String() + " to quit the game.\r\n\r\n"
	ret += "                      __________________________\r\n" + "                     |OFFo oON                  |\r\n" + "                     | .----------------------. |\r\n" + "                     | |  .----------------.  | |\r\n" + "                     | |  |                |  | |\r\n" + "                     | |))|                |  | |\r\n" + "                     | |  |                |  | |\r\n" + "                     | |  |                |  | |\r\n" + "                     | |  |                |  | |\r\n" + "                     | |  |                |  | |\r\n" + "                     | |  |                |  | |\r\n" + "                     | |  '----------------'  | |\r\n" + "                     | |__GAME BOY____________/ |\r\n" + "    Keyboard:Up↑ <--------+     ________        |\r\n" + "                     |    +    (Nintendo)       |\r\n" + "                     |  _| |_   \"\"\"\"\"\"\"\"   .-.  |\r\n" + "  Keyboard:Left← <----+[_   _]---+    .-. ( +---------> Keyboard:X\r\n" + "                     |   |_|     |   (   ) '-'  |\r\n" + "                     |    +      |    '-+   A   |\r\n" + "  Keyboard:Down↓ <--------+ +----+     B+-------------> Keyboard:Z\r\n" + "                     |      |   ___   ___       |\r\n" + "                     |      |  (___) (___)  ,., |\r\n" + "Keyboard:Right→ <-----------+ select st+rt ;:;: |\r\n" + "                     |           +     |  ,;:;' /\r\n" + "                  jgs|           |     | ,:;:'.'\r\n" + "                     '-----------------------`\r\n" + "                                 |     |\r\n" + "           Keyboard:Backspace <--+     +-> Keyboard:Enter\r\n"
	// Clean screen
	_, err := player.Conn.Write([]byte("\033[2J\033[H" + ret))
	if err != nil {
		return -1
	}
	for {
		buf := make([]byte, 512)
		n, err := player.Conn.Read(buf)
		inputKey := buf[:n]
		if err != nil {
			return -1
		}

		// Enter key pressed
		if inputKey[len(inputKey)-1] == 0 || inputKey[len(inputKey)-1] == 10 {
			return 1
		}
	}
}

func (player *Player) Logout() {
	// Disconnect serial port
	if player.Emulator.Serial.Target != nil {
		player.Emulator.Serial.Target.Target = nil
	}

	playerIndex := 0
	for k, v := range PlayerList {
		if v.ID == player.ID {
			playerIndex = k
			break
		}
	}

	if playerIndex != 0 {
		PlayerList = append(PlayerList[:playerIndex], PlayerList[playerIndex+1:]...)
	}
}

func (player *Player) Serve() {

	game := player.Welcome()
	fmt.Println("gameindex: ", game)//testline
	

	if game < 0 {
		log.Println("User quit")
		player.Logout()
		return
	}

	if player.Instruction() < 0 {
		log.Println("User quit")
		player.Logout()
		return
	}

	// Set the display driver to TELNET
	go player.Emulator.DisplayDriver.Run(player.Emulator.DrawSignal, func() {})
	player.Emulator.Init((*player.GameList)[player.Selected].Path)
	go player.Emulator.Run()

	for {
		buf := make([]byte, 512)
		n, err := player.Conn.Read(buf)
		if err != nil {
			log.Println("Error reading", err.Error())
			player.Emulator.Exit = true
			player.Logout()
			return
		}
		// If "Q" was pressed ,close the connection
		if buf[n-1] == 113 {
			log.Println("User quit")
			player.Emulator.Exit = true
			err := player.Conn.Close()
			if err != nil {
				log.Println("Failed to close connection")
			}
			player.Logout()
			return
		}
		// Handle user input
		player.Emulator.Controller.NewInput(buf[:n])
	}
}
