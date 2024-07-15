var editor_content = {}

var ENTRY_STRUCT = {

    entry:[
        {
            "title":"",
            "id":"",
            "type":""
        }
    ]

}

var PEERS_ENTRY_STRUCT = {

    room_name:[]

}


async function getContentList(){

  let resp = await fetch("/api/content/entry", {
    method: "GET"
  })

  let result = await resp.json()

  if(result.status != "success"){

    alert("failed to get sample list")

    return

  }


  let contentReader = document.getElementById("content-reader")

  let contentEntry = JSON.parse(result.reply)

  if (contentEntry.entry == null){

    contentReader.innerHTML = `
        <pre> :(    Nothing to see here, yet </pre>
    `


  } else {

    for(let i = 0; i < contentEntry.entry.length; i ++){

        contentReader.innerHTML += `
        <a href="/content/${contentEntry.entry[i].type}/${contentEntry.entry[i].id}">
            ${contentEntry.entry[i].title}
        </a>
        <br>
        `
     
    }
  }


}



async function getRoomList(){

    let resp = await fetch("/api/peers/entry", {
      method: "GET"
    })
  
    let result = await resp.json()
  
    if(result.status != "success"){
  
      alert("failed to get sample list")
  
      return
  
    }
  
  
    let roomReader = document.getElementById("room-reader")
  
    let roomEntry = JSON.parse(result.reply)
  
    if (roomEntry.room_name == null){
  
      roomReader.innerHTML = `
          <pre> :(     You're not invited, yet </pre>
      `
  
  
    } else {
  
      for(let i = 0; i < roomEntry.room_name.length; i ++){
  
          roomReader.innerHTML += `
          <a href="/room/${roomEntry.room_name[i]}">
              ${roomEntry.room_name[i]}
          </a>
          <br>
          `
       
      }
    }
  
  
  }

(async function() {

    await getContentList()

    await getRoomList()
 
 })()