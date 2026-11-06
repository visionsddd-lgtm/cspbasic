10 ' ===== CSP-ZORK: a tiny adventure for CSP BASIC =====
20 DIM TEMPLATE THING@(KEY$, ALT$, NAME$, LOC%, PORT%, BOX%, OPEN%, LIT%, TEXT$)
30 NOBJ% = 9 : NROOM% = 5
40 DIM ITEM@(9) AS THING@
50 DIM RNAME$(5), RDESC$(5), DARK%(5), SEEN%(5), ENTER$(5), EXIT%(5, 6)
60 FOR R = 1 TO NROOM% : READ RNAME$(R), RDESC$(R), DARK%(R) : FOR D = 1 TO 6 : READ EXIT%(R, D) : NEXT D : NEXT R
70 FOR I = 1 TO NOBJ% : READ ITEM@(I) : NEXT I
80 CNDL% = 4 : SWCH% = 5 : MTCH% = 7 : TRAP% = 8 : EGG% = 9
90 ' Game rules stored as data and run with FN APPLY
100 CANSEE$ = "(R%) => DARK%(R%) = 0 OR LIGHT% = 1"
110 ENTER$(2) = "IF SEEN%(2) = 0 THEN PRINT \"A grandfather clock chimes somewhere deep in the house.\" : SEEN%(2) = 1"
120 ENTER$(5) = "IF SEEN%(5) = 0 THEN PRINT \"Your candle throws long shadows. Something scuttles away from the light.\" : SEEN%(5) = 1"
130 CHAN #1, 0 : CHAN #2, 0
140 SPAWN 9000
150 CLS : COLOR 14 : PRINT "C S P - Z O R K" : COLOR 7
160 PRINT "An old house, a dark cellar, and something hungry. Type HELP for commands."
170 ROOM% = 1 : GOSUB 1000
300 ' --- Main loop: act on a command, then give the clock fiber its turn ---
310 PRINT
320 INPUT "> "; CMD$
330 GOSUB 3000
340 IF V$ = "" THEN BLANK% = BLANK% + 1 : IF BLANK% > 5 THEN GOTO 900 ELSE PRINT "Pardon?" : GOTO 310
345 BLANK% = 0
350 GOSUB 4000
360 IF QUIT% = 1 THEN GOTO 900
370 TURN% = TURN% + 1 : SEND #1, TURN% : RECV #2, EV$
380 IF EV$ <> "" THEN PRINT EV$
390 IF DEAD% = 1 THEN GOTO 950
400 IF ROOM% = 1 AND ITEM@(EGG%).LOC% = 99 THEN GOTO 960
410 GOTO 310
900 PRINT "Goodbye." : GOTO 990
950 PRINT : COLOR 12 : PRINT "*** You have died ***" : GOTO 990
960 PRINT : PRINT "The egg glitters in the sunlight. You have recovered the treasure!"
970 COLOR 10 : PRINT "*** You have won in "; TURN%; " turns ***"
990 END

1000 ' --- Describe the current room ---
1010 GOSUB 1900
1020 PRINT : COLOR 15 : PRINT RNAME$(ROOM%) : COLOR 7
1030 IF NOT FN APPLY%(CANSEE$, ROOM%) THEN PRINT "It is pitch black. You are likely to be eaten by a grue." : RETURN
1040 PRINT RDESC$(ROOM%)
1050 FOR J = 1 TO NOBJ%
1060 IF ITEM@(J).LOC% = ROOM% AND ITEM@(J).PORT% < 2 THEN GOSUB 1100
1070 NEXT J
1080 FN APPLY(ENTER$(ROOM%))
1090 RETURN
1100 PRINT "There is "; ITEM@(J).NAME$; " here";
1110 IF ITEM@(J).BOX% = 0 THEN PRINT "." : RETURN
1120 IF ITEM@(J).OPEN% = 0 THEN PRINT " (closed)." : RETURN
1130 GOSUB 1200
1140 IF C$ = "" THEN PRINT " (open, empty)." ELSE PRINT " (open). It contains "; C$; "."
1150 RETURN
1200 ' List the contents of container J into C$
1210 C$ = ""
1220 FOR K = 1 TO NOBJ%
1230 IF ITEM@(K).LOC% = 100 + J THEN IF C$ = "" THEN C$ = ITEM@(K).NAME$ ELSE C$ = C$ + " and " + ITEM@(K).NAME$
1240 NEXT K
1250 RETURN
1300 ' --- Inventory ---
1310 PRINT "You are carrying:" : F% = 0
1320 FOR J = 1 TO NOBJ%
1330 IF ITEM@(J).LOC% = 99 THEN GOSUB 1360
1340 NEXT J
1350 IF F% = 0 THEN PRINT "  Nothing at all."
1355 RETURN
1360 F% = 1 : PRINT "  "; ITEM@(J).NAME$;
1370 IF ITEM@(J).LIT% = 1 THEN PRINT " (burning)";
1380 PRINT : RETURN
1900 ' Is a lit candle within reach?
1910 LIGHT% = 0 : IF ITEM@(CNDL%).LIT% = 1 AND FN HERE%(CNDL%) = 1 THEN LIGHT% = 1
1920 RETURN

3000 ' --- Split the command into verb V$ and noun N$ ---
3010 U$ = UCASE$(TRIM$(CMD$)) : V$ = "" : N$ = ""
3020 IF U$ = "" THEN RETURN
3030 SP% = INSTR(U$, " ")
3040 IF SP% = 0 THEN V$ = U$ : RETURN
3050 V$ = LEFT$(U$, SP% - 1) : N$ = TRIM$(MID$(U$, SP% + 1))
3060 IF LEFT$(N$, 4) = "THE " THEN N$ = MID$(N$, 5)
3070 RETURN

4000 ' --- Verb dispatch: every handler ends in RETURN back to the main loop ---
4010 IF V$ = "GO" THEN V$ = N$ : N$ = ""
4020 IF V$ = "NORTH" OR V$ = "SOUTH" OR V$ = "EAST" OR V$ = "WEST" OR V$ = "UP" OR V$ = "DOWN" THEN V$ = LEFT$(V$, 1)
4030 IF LEN(V$) = 1 AND INSTR("NSEWUD", V$) > 0 THEN GOTO 5000
4040 IF V$ = "LOOK" AND N$ <> "" THEN V$ = "X" : IF LEFT$(N$, 3) = "AT " OR LEFT$(N$, 3) = "IN " THEN N$ = MID$(N$, 4)
4050 IF V$ = "L" OR V$ = "LOOK" THEN GOTO 1000
4060 IF V$ = "I" OR V$ = "INV" OR V$ = "INVENTORY" THEN GOTO 1300
4070 IF V$ = "HELP" THEN GOTO 4900
4080 IF V$ = "QUIT" OR V$ = "Q" THEN QUIT% = 1 : RETURN
4090 IF V$ = "XYZZY" THEN PRINT "A hollow voice says ""Fool.""" : RETURN
4100 IF V$ = "GET" THEN V$ = "TAKE"
4110 IF V$ = "EXAMINE" THEN V$ = "X"
4120 IF V$ = "TURN" AND LEFT$(N$, 3) = "ON " THEN V$ = "LIGHT" : N$ = MID$(N$, 4)
4130 IF V$ = "PRESS" OR V$ = "FLIP" OR V$ = "PUSH" THEN V$ = "LIGHT"
4140 IF N$ = "LIGHTS" THEN N$ = "LIGHT"
4150 IF INSTR(" TAKE DROP OPEN CLOSE READ LIGHT X ", " " + V$ + " ") = 0 THEN PRINT "I don't know how to "; LCASE$(V$); "." : RETURN
4160 IF N$ = "" THEN PRINT "What do you want to "; LCASE$(V$); "?" : RETURN
4170 OB% = FN FINDOBJ%(N$)
4180 IF OB% = 0 THEN PRINT "I don't know what that is." : RETURN
4190 GOSUB 1900
4200 IF FN HERE%(OB%) = 0 THEN PRINT "You don't see that here." : RETURN
4210 IF NOT FN APPLY%(CANSEE$, ROOM%) AND ITEM@(OB%).LOC% <> 99 AND OB% <> SWCH% THEN PRINT "It's too dark to see!" : RETURN
4220 IF V$ = "TAKE" THEN GOTO 6000
4230 IF V$ = "DROP" THEN GOTO 6100
4240 IF V$ = "OPEN" THEN GOTO 6200
4250 IF V$ = "CLOSE" THEN GOTO 6300
4260 IF V$ = "READ" OR V$ = "X" THEN GOTO 6400
4270 IF V$ = "LIGHT" THEN GOTO 6500
4280 RETURN
4900 PRINT "Try: LOOK, N/S/E/W/U/D, TAKE, DROP, OPEN, CLOSE, READ, EXAMINE (X),"
4910 PRINT "     LIGHT, TURN ON, INVENTORY (I) and QUIT."
4920 RETURN

5000 ' --- Movement ---
5010 D% = INSTR("NSEWUD", V$)
5020 NR% = EXIT%(ROOM%, D%)
5030 IF NR% = 0 THEN PRINT "You can't go that way." : RETURN
5040 IF ROOM% = 4 AND D% = 6 AND ITEM@(TRAP%).OPEN% = 0 THEN PRINT "The trap door is closed." : RETURN
5050 ROOM% = NR%
5060 GOTO 1000

6000 ' --- TAKE / DROP ---
6010 IF ITEM@(OB%).LOC% = 99 THEN PRINT "You already have it." : RETURN
6020 IF ITEM@(OB%).PORT% <> 1 THEN PRINT "You can't take that." : RETURN
6030 ITEM@(OB%).LOC% = 99 : PRINT "Taken."
6040 RETURN
6100 IF ITEM@(OB%).LOC% <> 99 THEN PRINT "You aren't carrying that." : RETURN
6110 ITEM@(OB%).LOC% = ROOM% : PRINT "Dropped."
6120 RETURN
6200 ' --- OPEN / CLOSE ---
6210 IF ITEM@(OB%).BOX% = 0 THEN PRINT "You can't open that." : RETURN
6220 IF ITEM@(OB%).OPEN% = 1 THEN PRINT "It's already open." : RETURN
6230 ITEM@(OB%).OPEN% = 1
6240 IF OB% = TRAP% THEN PRINT "The trap door creaks open, revealing a rickety staircase down." : RETURN
6250 J = OB% : GOSUB 1200
6260 IF C$ = "" THEN PRINT "Opened. It's empty." ELSE PRINT "Opening the "; LCASE$(ITEM@(OB%).KEY$); " reveals "; C$; "."
6270 RETURN
6300 IF ITEM@(OB%).BOX% = 0 THEN PRINT "You can't close that." : RETURN
6310 IF ITEM@(OB%).OPEN% = 0 THEN PRINT "It's already closed." : RETURN
6320 ITEM@(OB%).OPEN% = 0 : PRINT "Closed."
6330 RETURN
6400 ' --- READ / EXAMINE ---
6410 PRINT ITEM@(OB%).TEXT$
6420 IF OB% = CNDL% AND ITEM@(CNDL%).LIT% = 1 THEN PRINT "It is burning steadily."
6430 IF ITEM@(OB%).BOX% = 1 AND ITEM@(OB%).OPEN% = 1 THEN J = OB% : GOSUB 1200 : IF C$ <> "" THEN PRINT "Inside you see "; C$; "."
6440 RETURN
6500 ' --- LIGHT / TURN ON ---
6510 IF OB% = SWCH% THEN GOTO 6600
6520 IF OB% <> CNDL% THEN PRINT "You can't light that." : RETURN
6530 IF SPENT% = 1 THEN PRINT "Only a puddle of wax remains." : RETURN
6540 IF ITEM@(CNDL%).LIT% = 1 THEN PRINT "It's already burning." : RETURN
6550 IF ITEM@(CNDL%).LOC% <> 99 THEN PRINT "You should pick it up first." : RETURN
6560 IF ITEM@(MTCH%).LOC% <> 99 THEN PRINT "You have nothing to light it with." : RETURN
6570 ITEM@(CNDL%).LIT% = 1 : PRINT "You strike a match. The candle flickers to life."
6580 IF DARK%(ROOM%) = 1 THEN GOTO 1000
6590 RETURN
6600 IF DARK%(3) = 0 THEN PRINT "The lights are already on." : RETURN
6610 DARK%(3) = 0 : PRINT "You fumble along the wall and find a switch. Click!"
6620 GOTO 1000

7000 ' --- Rooms: name, description, dark, then exits N S E W U D ---
7010 DATA "Front Porch", "You stand on the sagging porch of an old white house. The front door hangs open to the north.", 0, 2, 0, 0, 0, 0, 0
7020 DATA "Entrance Hall", "A dusty hall with peeling wallpaper. Doorways lead east and west, and the porch is back south.", 0, 0, 1, 3, 4, 0, 0
7030 DATA "Study", "Shelves of mouldering books surround an old writing desk. The hall is west.", 1, 0, 0, 0, 2, 0, 0
7040 DATA "Kitchen", "A cold kitchen that smells of old onions. A trap door is set into the floor. The hall is east.", 0, 0, 0, 2, 0, 0, 5
7050 DATA "Cellar", "A damp, low cellar with a stone shelf cut into one wall. Stairs lead up.", 1, 0, 0, 0, 0, 4, 0
7100 ' --- Objects (one THING@ record each): LOC% 99 = carried, 100+n = inside object n; PORT% 2 = scenery ---
7110 DATA "MAILBOX", "BOX", "a small mailbox", 1, 0, 1, 0, 0, "A dented tin mailbox with a little red flag."
7120 DATA "LEAFLET", "PAPER", "a leaflet", 101, 1, 0, 0, 0, "WELCOME TO CSP-ZORK! Treasure lies beneath the kitchen, but the cellar is dark and something down there is hungry. Matches are kept in the study desk - feel for the light switch by the door. Bring the treasure back to this porch to win."
7130 DATA "CHEST", "OAK CHEST", "an oak chest", 2, 0, 1, 0, 0, "Solid oak bound with iron straps. Far too heavy to move."
7140 DATA "CANDLE", "WAX", "a beeswax candle", 103, 1, 0, 0, 0, "A thick beeswax candle with a fresh wick."
7150 DATA "SWITCH", "LIGHT", "a light switch", 3, 2, 0, 0, 0, "An old brass toggle switch beside the door."
7160 DATA "DRAWER", "DESK", "a desk drawer", 3, 0, 1, 0, 0, "A shallow drawer in the writing desk."
7170 DATA "MATCHES", "MATCHBOOK", "a book of matches", 106, 1, 0, 0, 0, "A matchbook from the Flood Control Dam #3 gift shop."
7180 DATA "TRAP DOOR", "TRAPDOOR", "a trap door", 4, 2, 1, 0, 0, "A heavy wooden trap door with an iron ring."
7190 DATA "EGG", "JEWELED EGG", "a jewel-encrusted egg", 5, 1, 0, 0, 0, "A golden egg crusted with diamonds and sapphires. Exquisite!"

8000 ' --- FN FINDOBJ%: object number for a noun, or 0 ---
8010 DEF FN FINDOBJ%(K$)
8020 LOCAL J
8030 FOR J = 1 TO NOBJ%
8040 IF ITEM@(J).KEY$ = K$ OR ITEM@(J).ALT$ = K$ THEN RETURN J
8050 NEXT J
8060 RETURN 0
8070 END FN
8100 ' --- FN HERE%: 1 if the object is carried, in this room, or inside an open container that is ---
8110 DEF FN HERE%(OBN)
8120 LOCAL L
8130 L = ITEM@(OBN).LOC%
8140 IF L = 99 OR L = ROOM% THEN RETURN 1
8150 IF L > 100 AND ITEM@(L - 100).OPEN% = 1 THEN RETURN FN HERE%(L - 100)
8160 RETURN 0
8170 END FN

9000 ' --- Dungeon clock: its own fiber, woken once per turn over channel #1 ---
9010 SHARED DEAD%, SPENT%  ' anything else this fiber writes stays private to it
9020 BURN% = 0 : GLOOM% = 0
9030 RECV #1, T$
9040 IF EOF$ = "1" THEN RETURN
9050 MSG$ = ""
9060 IF ITEM@(CNDL%).LIT% = 1 THEN GOSUB 9200
9070 GOSUB 1900
9080 IF FN APPLY%(CANSEE$, ROOM%) THEN GLOOM% = 0 ELSE GOSUB 9300
9090 SEND #2, MSG$
9100 GOTO 9030
9200 BURN% = BURN% + 1
9210 IF BURN% = 10 THEN MSG$ = "Your candle is burning low."
9220 IF BURN% < 15 THEN RETURN
9230 ITEM@(CNDL%).LIT% = 0 : ITEM@(CNDL%).TEXT$ = "A sad little stub of wax." : SPENT% = 1
9240 MSG$ = "Your candle sputters and goes out."
9250 RETURN
9300 GLOOM% = GLOOM% + 1
9310 IF GLOOM% = 2 THEN MSG$ = "You hear a soft slithering in the darkness nearby..."
9320 IF GLOOM% >= 3 THEN DEAD% = 1 : MSG$ = "Oh no! A lurking grue slithers out of the dark and devours you!"
9330 RETURN
