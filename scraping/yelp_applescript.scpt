on run argv
  if (count of argv) < 5 then
    display dialog "Usage: osascript yelp_applescript.scpt \"Name\" \"City\" \"Country\" \"HotelID\" \"OutputPath\" [\"DirectURL\"]"
    return
  end if

  set hotelName to item 1 of argv
  set city to item 2 of argv
  set country to item 3 of argv
  set hotelID to item 4 of argv
  set outputPath to item 5 of argv

  set targetURL to ""
  if (count of argv) = 6 then
    set targetURL to item 6 of argv
  end if

  set pageType to "unknown"
  set pageSource to ""

  tell application "Safari"
    activate

    if (count of windows) is 0 then
      make new window
    end if

    set frontWin to front window
    set currentTab to current tab of frontWin

    if targetURL is not "" and targetURL begins with "http" then
      set URL of currentTab to targetURL
      set pageType to "hotel"
    else
      set searchURL to "https://www.yelp.com/search?find_desc=" & hotelName & "&find_loc=" & city & "," & country
      set URL of currentTab to searchURL
      set pageType to "search"
    end if

    delay 5

    set pageSource to do JavaScript "document.documentElement.outerHTML" in currentTab

    tell frontWin
      if (count of tabs) > 1 then
        close (tabs 2 thru -1)
      end if
    end tell
  end tell

  try
    set fileRef to open for access (POSIX file outputPath) with write permission
    write pageSource to fileRef
    close access fileRef
    display notification pageType & " page saved for " & hotelName with title "Yelp Scraper"
  on error errMsg
    display dialog "Save error: " & errMsg
  end try
end run
