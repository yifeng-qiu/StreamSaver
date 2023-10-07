#!/bin/sh
streamsaver&

if [ -e "/media/download/youtube.com/jawed//NA-Me at the zoo.mp4" ]
then
    rm "/media/download/youtube.com/jawed//NA-Me at the zoo.mp4"
fi

curl -iX POST http://localhost:1718/new -F "url=https://www.youtube.com/watch?v=jNQXAC9IVRw"

sleep 5

if [ -e "/media/download/youtube.com/jawed//NA-Me at the zoo.mp4" ]; then echo "success" ; fi


