SELECT DISTINCT h.city, h.country FROM hotels h LEFT JOIN hotel_reviews hr ON hr.hotel_id = h.hotel_id WHERE hr.hotel_id IS NULL ORDER BY h.city;


SELECT h.city, COUNT(*) AS hotel_count FROM hotels h LEFT JOIN hotel_reviews hr ON hr.hotel_id = h.hotel_id WHERE hr.hotel_id IS NULL GROUP BY h.city ORDER BY h.city;