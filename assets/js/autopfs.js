class Job {
    /**
     *
     * @param {string} JobId
     * @param {string} State
     * @param {Session[]} Sessions
     * @param {Message[]} Messages
     * @param {Date} JobDate
     * @param {Character[]} Characters
     * @constructor
     */
    constructor(JobId, State, Sessions, Messages, JobDate, Characters) {
        this.JobId = JobId;
        this.State = State;
        this.JobDate = new Date(JobDate);

        this.Sessions = Sessions;
        this.Messages = Messages;
        this.Characters = Characters;
    }

    /**
     * @param {Object} object
     * @return {Job}
     */
    static fromObject(object) {
        return new Job(
            object["JobId"],
            object["State"],
            object["Sessions"].map(Session.fromObject),
            object["Messages"].map(Message.fromObject),
            new Date(object["JobDate"]),
            object["Characters"].map(Character.fromObject),
        );
    }
}

class Session {
    /**
     * @param {Date} Date
     * @param {number[]} EventNumber
     * @param {string} Game
     * @param {number} Season
     * @param {number} Number
     * @param {string} Variant
     * @param {string} ScenarioName
     * @param {number[]} Character
     * @param {boolean} Player
     * @param {boolean} GM
     * @constructor
     */
    constructor(Date, EventNumber, Game, Season, Number, Variant, ScenarioName, Character, Player, GM) {
        this.Date = Date;
        this.EventNumber = EventNumber;
        this.Game = Game;
        this.Season = Season;
        this.Number = Number;
        this.Variant = Variant;
        this.ScenarioName = ScenarioName;
        this.Character = Character;
        this.Player = Player;
        this.GM = GM;
    }

    /**
     *
     * @param {Object} object
     * @return {Session}
     */
    static fromObject(object) {
        let characters = [];
        if (object["Character"] !== null) {
            characters = object["Character"].map(n => {
                const i = parseInt(n);
                if (isNaN(i)) {
                    console.log(`failed parsing character ${n} as integer`);
                    return n
                }
                return i
            });
        }

        let eventNumber = [];
        if (object["EventNumber"] !== null) {
            eventNumber = object["EventNumber"].map(n => {
                const i = parseInt(n);
                if (isNaN(i)) {
                    console.log(`failed parsing ${n} as integer`);
                    return n
                }
                return i
            });
        }

        return new Session(
            new Date(object["Date"]),
            eventNumber,
            object["Game"],
            parseInt(object["Season"]),
            parseInt(object["Number"]),
            object["Variant"],
            object["ScenarioName"],
            characters,
            object["Player"],
            object["GM"],
        )
    }
}

class Message {
    /**
     * @param {Date} Time
     * @param {string} Message
     * @param {string} State
     * @constructor
     */
    constructor(Time, Message, State) {
        this.Time = Time;
        this.Message = Message;
        this.State = State;
    }

    /**
     * @param {Object} object
     * @return {Message}
     */
    static fromObject(object) {
        return new Message(
            new Date(object["Time"]),
            object["Message"],
            object["State"],
        )
    }
}

class Character {
    /**
     * @param {string} System
     * @param {number} Number
     * @param {string} Name
     * @param {Object} Prestige
     * @param {string} Faction
     * @constructor
     */
    constructor(System, Number, Name, Prestige, Faction) {
        this.System = System;
        this.Number = Number;
        this.Name = Name;
        this.Prestige = Prestige;
        this.Faction = Faction;
    }

    /**
     *
     * @param {Object} object
     * @return {Character}
     */
    static fromObject(object) {
        return new Character(
            object["System"],
            parseInt(object["Number"]),
            object["Name"],
            object["Prestige"],
            object["Faction"]
        )
    }
}

/**
 * @return {string}
 */
function WsUrl() {
    const url = new URL(location.href);
    if (url.protocol === "https") {
        url.protocol = "wss"
    } else {
        url.protocol = "ws"
    }
    url.pathname += "/ws";

    console.log(url.href);
    return url.href;
}

/**
 * @return {boolean}
 */
function IsView() {
    const p = Param("view");
    return p !== "";
}

/**
 * @return {string}
 */
function Param(key) {
    const p = (new URL(location.href)).searchParams.get(key);
    if (p === null) {
        return ""
    }
    return p
}

let lastMessage = new Date(0);

function Status() {
    const messageList = document.getElementById("messageList");
    const jobState = document.getElementById("jobState");
    const ws = new WebSocket(WsUrl());
    ws.onmessage = ev => {
        const message = JSON.parse(ev.data);
        const messageDate = new Date(message["Time"]);
        if (lastMessage != null && messageDate <= lastMessage) {
            return;
        }
        lastMessage = messageDate;
        const pre = document.createElement("PRE");
        pre.textContent = messageDate.toTimeString() + ": " + message["Message"];
        const li = document.createElement("LI");
        li.appendChild(pre);
        messageList.appendChild(li);
        if (jobState.textContent !== message["State"]) {
            jobState.textContent = message["State"];
        }

        if (message["State"] === "done" && !IsView()) {
            console.log("should be done...");
            document.location = "/html?id=" + Param("id");
            ws.onclose = null;
            ws.close();
        }
    };
    ws.onerror = () => {
        ws.close();
    };
    ws.onclose = () => {
        setTimeout(Status, 1000);
    };
}

let job = null;
let sortColumn = "Date";
let sortAscend = true;

function Html() {
    BuildHeader();

    const JsonUrl = new URL(location.href);
    JsonUrl.pathname = "/json";
    JsonUrl.searchParams.set("id", Param("id"));
    fetch(JsonUrl.href).then(response => {
        return response.json();
    }).then(json => {
        job = Job.fromObject(json);
        Render(job);
    });
}

function BuildHeader() {
    const prevTableHeader = document.getElementById("jobTableHead");
    const table = prevTableHeader.parentNode;
    const tableHeader = document.createElement("THEAD");
    Columns.forEach(column => {
        const el = document.createElement("TH");
        const a = document.createElement("a");
        a.onclick = SortBy(column);
        a.innerText = column.Name;
        el.appendChild(a);
        tableHeader.appendChild(el);
    });
    table.insertBefore(tableHeader, prevTableHeader);
    table.removeChild(prevTableHeader);
}

/**
 *
 * @param {Column} column
 * @returns {Function}
 */
function SortBy(column) {
    return function () {
        if (sortColumn === column.Name) {
            sortAscend = !sortAscend
        } else {
            sortColumn = column.Name;
            sortAscend = true;
        }
        job.Sessions.sort((i, j) => {
            return sortAscend ? column.Compare(i, j, sortAscend) : column.Compare(j, i, sortAscend);
        });
        Render(job);
    }
}

/**
 * @param {string} Name
 * @param {function(Session): HTMLElement} Render
 * @param {function(Session,Session,[boolean]): number} Compare
 * @param {function(Session, string[]): boolean} Select
 * @param Gadget
 * @constructor
 */
function Column(Name, Render, Compare, Select, Gadget) {
    this.Name = Name;
    this.Render = Render;
    this.Compare = Compare;
}

const Columns = [
    new Column(
        "Date",
        session => {
            if (session.Date.getFullYear() === 0) {
                return document.createTextNode("(missing)")
            }

            const y = session.Date.getFullYear().toString();
            let m = session.Date.getMonth().toString();
            let d = session.Date.getDate().toString();
            if (m.length < 2) {
                m = `0${m}`
            }
            if (d.length < 2) {
                d = `0${d}`
            }
            return document.createTextNode(`${y}-${m}-${d}`);
        },
        (i, j) => {
            return i.Date.valueOf() - j.Date.valueOf();
        },
        (s, range) => {
            const sv = s.Date.valueOf();
            const min = parseInt(range[0]);
            const max = parseInt(range[1]);
            if (!isNaN(min) && sv < min) {
                return false;
            }
            if (!isNaN(max) && sv > max) {
                return false;
            }
            return true;
        }, null
    ),
    new Column(
        "System",
        session => {
            return document.createTextNode(session.Game)
        },
        (i, j) => {
            return i.Game.localeCompare(j.Game);
        },
        null, null
    ),
    new Column(
        "Event #",
        session => {
            return document.createTextNode(
                session
                    .EventNumber
                    .map(n => {
                        return isNaN(n) ? "(missing)" : n
                    })
                    .join(", ")
            );
        },
        (i, j, ascend) => {
            if (ascend) {
                // find lowest
                let iMin = null, jMin = null;
                i.EventNumber.forEach(n => {
                    if (iMin === null || n < iMin) {
                        iMin = n;
                    }
                });
                j.EventNumber.forEach(n => {
                    if (jMin === null || n < jMin) {
                        jMin = n;
                    }
                });
                return iMin - jMin;
            }

            // find highest
            let iMax = null, jMax = null;
            i.EventNumber.forEach(n => {
                if (iMax === null || n > iMax) {
                    iMax = n;
                }
            });
            j.EventNumber.forEach(n => {
                if (jMax === null || n > jMax) {
                    jMax = n;
                }
            });
            return iMax - jMax;
        }, null, null
    ),
    new Column(
        "Character",
        session => {
            let first = true;
            const span = document.createElement("SPAN");
            session.Character.forEach(c => {
                if (first) {
                    first = false;
                } else {
                    span.appendChild(document.createTextNode(", "));
                }

                span.appendChild(document.createTextNode(Math.abs(c).toString()));
                if (c < 0) {
                    const sup = document.createElement("SUP");
                    sup.innerText = "GM";
                    span.appendChild(sup);
                }
            });
            return span;
        },
        (i, j, ascend) => {
            if (ascend) {
                // find lowest
                let iMin = null, jMin = null;
                i.Character.forEach(n => {
                    n = n < 0 ? -1 * n : n;
                    if (iMin === null || n < iMin) {
                        iMin = n;
                    }
                });
                j.Character.forEach(n => {
                    n = n < 0 ? -1 * n : n;
                    if (jMin === null || n < jMin) {
                        jMin = n;
                    }
                });
                return iMin - jMin;
            }

            // find highest
            let iMax = null, jMax = null;
            i.Character.forEach(n => {
                n = n < 0 ? -1 * n : n;
                if (iMax === null || n > iMax) {
                    iMax = n;
                }
            });
            j.Character.forEach(n => {
                n = n < 0 ? -1 * n : n;
                if (jMax === null || n > jMax) {
                    jMax = n;
                }
            });
            return iMax - jMax;
        }, null, null
    ),
    new Column(
        "Season",
        session => {
            return document.createTextNode(session.Season.toString());
        },
        (i, j) => {
            return i.Season - j.Season;
        }, null, null,
    ),
    new Column(
        "Number",
        session => {
            return document.createTextNode(session.Number.toString());
        },
        (i, j) => {
            return i.Number - j.Number;
        }, null, null,
    ),
    new Column(
        "Variant",
        session => {
            return document.createTextNode(session.Variant);
        },
        (i, j) => {
            return i.Variant.localeCompare(j.Variant);
        }, null, null,
    ),
    new Column(
        "Scenario Name",
        session => {
            return document.createTextNode(session.ScenarioName);
        },
        (i, j) => {
            return i.ScenarioName.localeCompare(j.ScenarioName);
        }, null, null,
    ),
    new Column(
        "Player/GM",
        session => {
            let s = "P";
            if (session.GM) {
                s = "GM";
                if (session.Player) {
                    s = "P/GM";
                }
            }
            return document.createTextNode(s);
        },
        (i, j) => {
            iVal = (i.Player ? 1 : 0) + (i.GM ? 2 : 0);
            jVal = (j.Player ? 1 : 0) + (j.GM ? 2 : 0);
            return iVal - jVal;
        },
        null, null,
    ),
];

/**
 * @param {Job} job
 */
function Render(job) {
    const prevTbody = document.getElementById("jobTableBody");
    const table = prevTbody.parentNode;
    table.removeChild(prevTbody);

    const tbody = document.createElement("TBODY");
    tbody.id = "jobTableBody";

    job.Sessions.forEach(session => {
        const row = document.createElement("TR");
        Columns.forEach(column => {
            const cell = document.createElement("TD");
            cell.appendChild(column.Render(session));
            row.appendChild(cell);
        });
        tbody.appendChild(row);
    });

    table.appendChild(tbody);
}