on run argv
    if (count of argv) < 4 then
        display dialog "Usage: osascript script.scpt \"Hotel Name\" \"City\" \"Country\" \"HotelID\""
        return
    end if

    set hotelName to item 1 of argv
    set city to item 2 of argv
    set country to item 3 of argv
    set hotelID to item 4 of argv

    -- Quit Safari if running to ensure clean state
    tell application "Safari"
        quit
    end tell
    delay 2 -- Wait for Safari to quit

    tell application "Safari"
        activate

        -- Close extra tabs to start clean
        try
            repeat with w in windows
                repeat with t in tabs of w
                    if (count of tabs of w) > 1 then
                        close t
                    end if
                end repeat
            end repeat
        end try

        -- Build search query and URL
        set safeHotelName to my replaceText(hotelName, "'", "")
        set safeCity to my replaceText(city, "'", "")
        set safeCountry to my replaceText(country, "'", "")
        set searchQuery to safeHotelName & "+" & safeCity & "+" & safeCountry
        set searchURL to "https://www.tripadvisor.com/Search?q=" & searchQuery

        -- Open TripAdvisor search directly
        make new document with properties {URL:searchURL}
        set theDoc to document 1

        -- Wait briefly for search results
        delay 5

        -- Click the first hotel result
        do JavaScript "var links = document.querySelectorAll('a[href*=\"/Hotel_Review-\"]'); if (links.length > 0) { links[0].click(); }" in theDoc

        -- Wait for hotel page and find the hotel document
        delay
        set found to false
        repeat with d in documents
            if URL of d contains "Hotel_Review" then
                set theDoc to d
                set found to true
                exit repeat
            end if
        end repeat
        if not found then
            set theDoc to document 1
        end if

        -- Click "All Reviews" or similar (adjust selector, e.g., for "Show all reviews")
        do JavaScript "var links = document.querySelectorAll('a'); for (var i = 0; i < links.length; i++) { if (links[i].textContent.includes('All Reviews') || links[i].href.includes('#REVIEWS')) { links[i].click(); break; } }" in theDoc

        -- Wait for reviews to load and find the reviews document
        delay 3
        set found to false
        repeat with d in documents
            if URL of d contains "#REVIEWS" or URL of d contains "Hotel_Review" then
                set theDoc to d
                set found to true
                exit repeat
            end if
        end repeat
        if not found then
            set theDoc to document 1
        end if

        -- Get full page source
        set pageSource to do JavaScript "document.documentElement.outerHTML;" in theDoc

        -- Debug: check if page source is captured
        --display dialog "Page source length: " & (length of pageSource)

    end tell

    try
        -- Sanitize hotel name for filename (replace spaces with underscores)
        set sanitizedHotelName to my replaceText(hotelName, " ", "_")

        -- Save to relative directory: ./hotelReviewsSaved/City,Country/
        set folderPath to "./hotelReviewsSaved/" & city & "," & country & "/"

        -- Create folder if needed using shell
        do shell script "mkdir -p " & quoted form of folderPath

        set fileName to hotelID & "_tripadvisor_" & sanitizedHotelName & ".html"
        set fullPath to folderPath & fileName

    -- Save page source using AppleScript file writing to avoid shell issues

        set fileRef to open for access (POSIX file fullPath) with write permission
        write pageSource to fileRef
        close access fileRef
    on error errMsg
        display dialog "Save error: " & errMsg
    end try
end run

on replaceText(theText, searchString, replacementString)
    set oldDelimiters to AppleScript's text item delimiters
    set AppleScript's text item delimiters to searchString
    set theText to text items of theText
    set AppleScript's text item delimiters to replacementString
    set theText to theText as string
    set AppleScript's text item delimiters to oldDelimiters
    return theText
end replaceText