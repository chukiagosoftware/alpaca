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

  tell application "Safari"
    activate

    if targetURL is not "" and targetURL begins with "http" then
    -- Stage 2: Open specific hotel page
      make new document with properties {URL:targetURL}
      set pageType to "hotel"
      delay 4
    else
    -- Stage 1: Perform search
      set searchURL to "https://www.yelp.com/search?find_desc=" & hotelName & "&find_loc=" & city & "," & country
      make new document with properties {URL:searchURL}
      set pageType to "search"
      delay 4
    end if

    set theDoc to document 1
    delay 4

    set pageSource to do JavaScript "document.documentElement.outerHTML" in theDoc
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