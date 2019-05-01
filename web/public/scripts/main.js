// -- GLOBALS -- //

var globs = {
    user: {},
    dir_out: document.getElementsByClassName('dir-content')[0],
    trace_out: document.getElementsByClassName('dir-trace')[0],
    cu_dir: "",
    item_ctx_menu: document.getElementsByClassName('item-context-menu')[0],
    item_inf_p_wr: document.getElementsByClassName('item-info-popup-wrapper')[0],
    dir_ctx_menu: document.getElementsByClassName('dir-context-menu')[0],
    file_up_wr: document.getElementsByClassName('file-upload-container-wrapper')[0],
    up_files: [],
    up_files_in: [],
    client_inf_wr: document.getElementsByClassName('client-info-container-wrapper')[0],
    u_clients: [],
    file_pre_wr: document.getElementsByClassName('file-preview-wrapper')[0],
};

// ------------- //

// -- FUNCTIONS -- //

function getUserCreds() {
    fetch('/api/creds')
        .then(res => {
            if (res.ok)
                return res.json();
            window.location.assign('/all_functions.html');
        })
        .then(res => {
            globs.user = res;
            reloadUser();
        });
}

function reloadUser() {
    if (!globs.user.username || !globs.user.email)
        return;
    document.getElementById('user-profile-username').innerHTML = globs.user.username;
}

function loadDir(d, cb = displayDir, dis_tr = true) {
    if (d === "")
        d = "root";

    fetch(`/api/req/dir?d=${d}`)
        .then(res => {
            if (res.ok)
                return res.json();
            loadDir('root');
        })
        .then(res => {
            if (!res)
                return;

            if (dis_tr)
                displayTrace(d);
            cb(d, res);
        });
}

function displayTrace(d, trgt = globs.trace_out) {
    if (d === "")
        d = "root";

    if (d !== "root")
        window.history.pushState(null, '', `/${d}`);
    else
        window.history.pushState(null, '', '/');

    fetch(`/api/req/trace?u=${d}`)
        .then(res => {
            if (res.ok)
                return res.json();
        })
        .then(res => {
            if (!res)
                return;
            trgt.innerHTML = '';

            res.reverse().forEach(e => {
                let o = document.createElement('div');
                o.classList.add('dir-trace-item');

                o.ondragover = ev => {
                    ev.preventDefault();
                };

                o.ondrop = ev => {
                    ev.preventDefault();
                    let df = JSON.parse(ev.dataTransfer.getData('application/json'));

                    if (df.UniqueId === e.UniqueId)
                        return false;

                    moveFile(df.UniqueId.split('_')[1], e.UniqueId.split('_')[1]);
                };

                o.onclick = () => {
                    loadDir(e.UniqueId.split('_')[1])
                };

                if (e.UniqueId.endsWith('_root')) {
                    let i = document.createElement('i');
                    i.classList.add('fas', 'fa-home');
                    o.appendChild(i);
                } else if (e.UniqueId.endsWith('_trash')) {
                    let i = document.createElement('i');
                    i.classList.add('fas', 'fa-trash-alt');
                    o.appendChild(i);
                } else {
                    o.innerHTML = e.Name;
                }

                trgt.appendChild(o);
                trgt.appendChild(document.createTextNode('/'));
            });
        });
}

function displayDir(d, dir, trgt = globs.dir_out) {
    globs.cu_dir = d;
    trgt.innerHTML = "";

    dir.forEach(f => {
        let pd = document.createElement('div');
        pd.classList.add('dir-item');
        pd.draggable = true;

        // if (f.MIMEType.startsWith('image/')) {
        //     let con = document.createElement('con');
        //     let img = document.createElement('div');
        //     img.style.width = "100%";
        //     img.style.height = "100%";
        //     downloadFile(f.UniqueId.split('_')[1], f, false, blb_url => {
        //         img.style.backgroundImage = `url(${blb_url})`;
        //         img.style.backgroundPosition = 'center';
        //         img.style.backgroundSize = 'cover';
        //         img.onload = () => {
        //             window.URL.revokeObjectURL(blb_url);
        //         };
        //     });
        //     con.classList.add('dir-item-preview');
        //     con.appendChild(img);
        //     pd.appendChild(con);
        // } else if (f.MIMEType.startsWith('video/')) {
        //     let con = document.createElement('div');
        //     con.style.position = 'relative';
        //     con.style.overflow = 'hidden';
        //     let vid = document.createElement('video');
        //     vid.autoplay = true;
        //     vid.muted = true;
        //     vid.loop = true;
        //     vid.style.width = "100%";
        //     vid.style.height = "auto";
        //     vid.style.position = 'absolute';
        //     vid.style.left = '0';
        //     vid.style.top = '0';
        //     downloadFile(f.UniqueId.split('_')[1], f, false, blb_url => {
        //         let src = document.createElement('source');
        //         src.src = blb_url;
        //         src.type=f.MIMEType;
        //         vid.appendChild(src);
        //         src.onload = () => {
        //             window.URL.revokeObjectURL(blb_url);
        //         };
        //     });
        //     vid.appendChild(document.createTextNode('Loading ... '));
        //     con.classList.add('dir-item-preview');
        //     con.appendChild(vid);
        //     pd.appendChild(con);
        // } else if (f.MIMEType.startsWith('text/') && f.MIMEType !== 'text/directory') {
        //     let con = document.createElement('div');
        //     con.style.overflow = 'hidden';
        //     let doc = document.createElement('pre');
        //     doc.style.fontSize = '8px';
        //     doc.style.backgroundColor = '#fff';
        //     doc.style.border = '.5px solid #f2f2f2';
        //     doc.style.width = '80%';
        //     doc.style.display = 'block';
        //     doc.style.wordBreak = 'break-all';
        //     doc.style.margin = '0 auto';
        //     downloadFile(f.UniqueId.split('_')[1], f, false, blb_url => {
        //         fetch(blb_url)
        //             .then(res => res.text())
        //             .then(res => {
        //                 window.URL.revokeObjectURL(blb_url);
        //                 doc.innerHTML = res;
        //             });
        //     });
        //     con.classList.add('dir-item-preview');
        //     con.appendChild(doc);
        //     pd.appendChild(con);
        // } 

        if (f.MIMEType.startsWith('image/')) {
            let con = document.createElement('con');
            let img = document.createElement('div');
            img.style.width = "100%";
            img.style.height = "100%";
            img.style.backgroundImage = `url(/media/image.png)`;
            img.style.backgroundPosition = 'center';
            img.style.backgroundSize = 'cover';
            con.classList.add('dir-item-preview');
            con.appendChild(img);
            pd.appendChild(con);
        } else if (f.MIMEType.startsWith('video/')) {
            let con = document.createElement('con');
            let img = document.createElement('div');
            img.style.width = "100%";
            img.style.height = "100%";
            img.style.backgroundImage = `url(/media/video.png)`;
            img.style.backgroundPosition = 'center';
            img.style.backgroundSize = 'cover';
            con.classList.add('dir-item-preview');
            con.appendChild(img);
            pd.appendChild(con);
        } else if (f.MIMEType.startsWith('application/')) {
            let con = document.createElement('con');
            let img = document.createElement('div');
            img.style.width = "100%";
            img.style.height = "100%";
            img.style.backgroundImage = `url(/media/application.png)`;
            img.style.backgroundPosition = 'center';
            img.style.backgroundSize = 'cover';
            con.classList.add('dir-item-preview');
            con.appendChild(img);
            pd.appendChild(con);
        } else if (f.MIMEType.startsWith('text/') && f.MIMEType !== 'text/directory') {
            let con = document.createElement('con');
            let img = document.createElement('div');
            img.style.width = "100%";
            img.style.height = "100%";
            img.style.backgroundImage = `url(/media/text.png)`;
            img.style.backgroundPosition = 'center';
            img.style.backgroundSize = 'cover';
            con.classList.add('dir-item-preview');
            con.appendChild(img);
            pd.appendChild(con);
        } else if (f.MIMEType === 'text/directory') {
            let con = document.createElement('div');
            con.style.overflow = 'hidden';
            con.style.backgroundImage = 'url(/media/folder.png)';
            con.style.backgroundSize = 'cover';
            con.style.opacity = '.5';
            con.style.filter = 'grayscale(5%)';
            con.classList.add('dir-item-preview');
            pd.appendChild(con);
        } else {
            let con = document.createElement('con');
            let img = document.createElement('div');
            img.style.width = "100%";
            img.style.height = "100%";
            img.style.backgroundImage = `url(/media/file.png)`;
            img.style.backgroundPosition = 'center';
            img.style.backgroundSize = 'cover';
            con.classList.add('dir-item-preview');
            con.appendChild(img);
            pd.appendChild(con);
        }

        let d = document.createElement('div');
        d.classList.add('dir-item-info');
        pd.appendChild(d);

        let i = document.createElement('i');
        i.classList.add('dir-item-icon');
        d.appendChild(i);

        d.appendChild(document.createTextNode(f.Name));
        d.setAttribute('u_id', f.UniqueId.split('_')[1]);

        pd.oncontextmenu = e => {
            e.preventDefault();
            displayContextMenu(e, f);
        };

        // let drel = document.createElement('div');
        // drel.classList.add('dir-item-dragging');
        // drel.innerHTML = d.innerHTML;

        pd.ondragstart = function (e) {
            // e.preventDefault();
            if (e.ctrlKey) {
                e.dataTransfer.dropEffect = 'copy';
            } else {
                e.dataTransfer.dropEffect = 'move';
            }
            e.dataTransfer.setData('application/json', JSON.stringify(f));
            // console.log(e.dataTransfer.getData('application/json'));
        };

        if (f.IsDir) {
            pd.classList.add('dir-item-dir');
            d.classList.add('marlx-dir');
            i.classList.add('far', 'fa-folder');

            pd.ondrop = e => {
                e.preventDefault();
                let df = JSON.parse(e.dataTransfer.getData('application/json'));

                if (df.UniqueId === f.UniqueId)
                    return false;

                moveFile(df.UniqueId.split('_')[1], f.UniqueId.split('_')[1]);
            };
            pd.ondragover = e => {
                e.preventDefault();
            };

            pd.ondblclick = () => {
                loadDir(d.getAttribute('u_id'));
            };
        } else {
            pd.classList.add('dir-item-file');
            d.classList.add('marlx-file');
            i.classList.add('far', 'fa-file-alt');

            pd.onclick = () => {
                showFilePreview(f);
            };

            pd.ondblclick = () => {
                downloadFile(d.getAttribute('u_id'), f);
            };
        }

        trgt.appendChild(pd);
    });
}

function showFilePreview(f) {
    globs.file_pre_wr.style.display = 'block';
    showModal('.file-preview-container', '.file-preview-wrapper');

    let trgt = document.getElementsByClassName('file-preview-out')[0];
    trgt.innerHTML = '';

    let con = document.createElement('div');
    con.style.width = '100%';
    
    con.style.padding = '0';
    con.style.marginTop = '15px';

    con.style.textAlign = 'center';
    con.style.fontSize = '1.25em';

    if (f.MIMEType.startsWith('image/')) {
        let img = document.createElement('img');
        downloadFile(f.UniqueId.split('_')[1], f, false, blb_url => {
            img.src = `${blb_url}`;
            img.onload = () => {
                window.URL.revokeObjectURL(blb_url);
            };
        });
        img.style.width = '100%';
        img.style.borderRadius = '5px';
        img.style.margin = '0';
        img.style.marginBottom = '7.5px';
        con.appendChild(img);
    } else if (f.MIMEType.startsWith('video/')) {
        let vid = document.createElement('video');
        downloadFile(f.UniqueId.split('_')[1], f, false, blb_url => {
            let src = document.createElement('source');
            src.src = `${blb_url}`;
            src.onload = () => {
                window.URL.revokeObjectURL(blb_url);
            };
            vid.appendChild(src);
            vid.appendChild(document.createTextNode('Error whilst loading video ... '));
        });

        vid.controls = true;
        vid.autoplay = true;

        vid.style.width = '100%';
        vid.style.borderRadius = '5px';
        vid.style.margin = '0';
        vid.style.marginBottom = '7.5px';

        con.appendChild(vid);
    } else if (f.MIMEType.startsWith('text/') && f.MIMEType !== 'text/directory') {
        let tcon = document.createElement('pre');
        tcon.classList.add('item-preview-text');

        if (f.Size > 1000000) {
            tcon.innerHTML = 'File too big to display a preview!';
            con.appendChild(tcon);
            con.appendChild(document.createTextNode(f.Name));
            trgt.appendChild(con);
            return;
        }

        downloadFile(f.UniqueId.split('_')[1], f, false, blb_url => {
            fetch(blb_url)
                .then(res => {
                    if (res.ok)
                        return res.text();
                })
                .then(res => {
                    window.URL.revokeObjectURL(blb_url);
                    if (!res)
                        tcon.innerHTML = '';
                    else
                        tcon.appendChild(document.createTextNode(res));
                });
        });

        con.appendChild(tcon);
    }

    con.appendChild(document.createTextNode(f.Name));
    trgt.appendChild(con);
}

function moveFile(fid, did) {
    fetch(`/api/mov/item`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                uniqueId: fid,
                targDir: did,
            }),
        })
        .then(res => {
            if (res.ok)
                return res.json();
        })
        .then(res => {
            if (!res)
                return
            loadDir(globs.cu_dir);
        });
}

function getCookie(key) {
    let cks = decodeURIComponent(document.cookie);
    let ck_ps = cks.split(';');

    for (let ck of ck_ps) {
        ck = ck.trim();
        if (ck.startsWith(key + '=')) {
            return ck.split('=')[1];
        }
    }
}

function setCookie(key, value) {
    document.cookie = `${key}=${value}`;
}

function downloadFile(f_id, f, prompt_dwn = true, cb = () => {}) {
    fetch(`/api/req/file?u=${f_id}`)
        .then(res => res.arrayBuffer())
        .then(res => {
            let data = new Uint8Array(res);

            let bts = marlx_crypto.hexUndigest(getCookie('marlx_scr'));
            let k = marlx_crypto.stringToBytes(getCookie('marlx_tkn').substr(0, 32));
            marlx_crypto.aesctrDec(bts, k).then(pwd => {
                pwd = marlx_crypto.bytesToString(pwd);
                let salt = f.Salt;

                marlx_crypto.hash(pwd, salt).then(h => {
                    let key = marlx_crypto.stringToBytes(h.substr(28));

                    marlx_crypto.aesctrDec(data, key).then(dec => {
                        let blb = new Blob([dec], {
                            type: f.MIMEType
                        });

                        let bur = window.URL.createObjectURL(blb);

                        if (prompt_dwn) {
                            let a = document.getElementById('item-download-a');
                            a.href = bur;
                            a.download = f.Name;
                            a.click();

                            window.URL.revokeObjectURL(bur);
                        }

                        cb(bur);
                    });
                });
            });
        });
}

function sideBarDropHandler(e, did) {
    e.preventDefault();
    let df = JSON.parse(e.dataTransfer.getData('application/json'));

    if (df.UniqueId.split('_')[1] === did)
        return false;

    moveFile(df.UniqueId.split('_')[1], did);
}

function loadSidebar(d) {
    let trgt = document.getElementById(`chevron-${d}`);
    if (!trgt)
        return;

    trgt.classList.toggle('fa-chevron-right');
    trgt.classList.toggle('fa-chevron-down');

    if (!trgt.classList.contains('fa-chevron-down')) {
        document.querySelector(`div[parent="${d}"]`).innerHTML = '';
        return;
    }

    loadDir(d, (x, dir) => {
        trgt = document.querySelector(`div[parent="${d}"]`);
        if (!trgt)
            return;

        dir.forEach(f => {
            if (!f.IsDir)
                return;

            let sp_id = f.UniqueId.split('_')[1];

            let o = document.createElement('div');
            o.classList.add('side-bar-item');
            o.onclick = () => {
                loadDir(sp_id);
            };

            o.ondrop = e => {
                sideBarDropHandler(e, f.UniqueId.split('_')[1]);
            };

            o.ondragover = e => {
                e.preventDefault();
            };

            o.id = `side-bar-item-${sp_id}`;

            o.style.paddingLeft = (+document.getElementById(`side-bar-item-${d}`).style.paddingLeft
                .substr(0, document.getElementById(`side-bar-item-${d}`).style.paddingLeft.length - 2) + 10) + "px";

            let i = document.createElement('i');
            i.classList.add('fas', 'fa-chevron-right');
            i.id = `chevron-${sp_id}`
            i.onclick = () => {
                loadSidebar(sp_id);
            };

            let con = document.createElement('span');
            con.classList.add('side-bar-item-content');
            con.innerHTML = f.Name;

            o.appendChild(i);
            o.appendChild(con);

            let os = document.createElement('div');
            os.classList.add('side-bar-item-sub');
            os.setAttribute('parent', sp_id);

            trgt.appendChild(o);
            trgt.appendChild(os);
        });
    }, false);
}

function deleteFile(uid, cb = () => {
    loadDir(globs.cu_dir);
}) {
    fetch(`/api/del/item`, {
            method: 'DELETE',
            body: JSON.stringify({
                uniqueId: uid,
            }),
        })
        .then(res => {
            if (res.ok)
                return res.json();
        })
        .then(res => {
            if (!res)
                return;
            cb();
        });
}

function displayContextMenu(e, f, ctx = globs.item_ctx_menu) {
    ctx.style.display = 'block';

    ctx.style.left = Math.min(e.clientX, window.innerWidth - ctx.offsetWidth - 15) + "px";
    ctx.style.top = Math.min(e.clientY, window.innerHeight - ctx.offsetHeight - 15) + "px";

    if (f.IsDir) {
        document.getElementById('c-m-i-download').classList.add('context-menu-item-disabled');
        document.getElementById('c-m-i-download').onclick = () => {};
    } else {
        document.getElementById('c-m-i-download').classList.remove('context-menu-item-disabled');
        document.getElementById('c-m-i-download').onclick = () => {
            downloadFile(f.UniqueId.split('_')[1], f);
        };
    }

    document.getElementById('c-m-i-delete').onclick = () => {
        deleteFile(f.UniqueId.split('_')[1]);
    };

    document.getElementById('c-m-i-rename').onclick = () => {
        let nname = window.prompt("Enter new name: ");
        if (nname === "" || nname === null)
            return;

        renameItem(f.UniqueId.split('_')[1], nname);
    };

    document.getElementById('c-m-i-details').onclick = () => {
        showItemInfo(f);
    };

    anime({
        targets: ctx,
        rotateX: 0,
        skewY: 0,
        opacity: 1,
        duration: 1000,
    });
}

function renameItem(uid, nname) {
    fetch('/api/upd/file', {
            method: 'PUT',
            body: JSON.stringify({
                uniqueId: uid,
                newName: nname,
            }),
        })
        .then(res => {
            if (res.ok)
                return res.json();
        })
        .then(res => {
            if (!res)
                return;
            loadDir(globs.cu_dir);
        });
}

function parseSize(s, type = 'short') {
    let sizes = {
        short: ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'],
        long: ['Byte', 'Kilo Byte', 'Mega Byte', 'Giga Byte', 'Tera Byte', 'Peta Byte', 'Exa Byte', 'Zetta Byte', 'Yotta Byte'],
    };

    if (!Object.keys(sizes).includes(type))
        type = 'short';

    if (typeof s !== "number")
        s = +s

    let i = 0;
    while (s >= 1000 && i < sizes[type].length) {
        s /= 1000;
        i++;
    }

    return Math.round(s * 100) / 100 + ' ' + sizes[type][i];
}

function parseDateTime(d, type = 'short') {
    let units = {
        short: ['ms', 's', 'min', 'h', 'day', 'week', 'year'],
        long: ['millisecond', 'second', 'minute', 'hour', 'day', 'week', 'year'],
    };
    let steps = [1000, 60, 60, 24, 7, 365];

    if (!Object.keys(units).includes(type))
        type = 'short';

    if (typeof d !== "string" && typeof d !== "number")
        return '';

    let dt = new Date(d);
    let diff = new Date().getTime() - dt.getTime();

    let i = 0;
    while (diff >= steps[i]) {
        diff /= steps[i];
        i++;
    }

    return Math.round(diff) + (type === 'short' ? '' : ' ') + units[type][i] +
        (Math.round(diff) > 1 && (type === 'long' || i >= 4) ? 's' : '') + ' ago';
}

function showItemInfo(f) {
    globs.item_inf_p_wr.style.display = 'block';

    document.getElementById('i-i-p-title').innerHTML = f.Name;
    document.getElementById('i-i-p-creation-time').innerHTML = parseDateTime(f.CreationTime, 'long');

    document.getElementById('i-i-p-parent-dir').innerHTML = f.ParentDir;
    document.getElementById('i-i-p-size').innerHTML = parseSize(f.Size);
    document.getElementById('i-i-p-mime-type').innerHTML = f.MIMEType;

    let tout = document.getElementById('i-i-p-clients');
    f.CTokens.forEach(ct => {
        let sp = document.createElement('span');
        sp.classList.add('marlx-function-link');
        sp.onclick = () => {
            showClientInfo(ct);
        };
        tout.appendChild(sp);
    });

    showModal('.item-info-popup', '.item-info-popup-wrapper');
}

function displayDirContextMenu(e, did = globs.cu_dir, ctx = globs.dir_ctx_menu) {
    ctx.style.display = 'block';

    ctx.style.left = Math.min(e.clientX, window.innerWidth - ctx.offsetWidth - 15) + "px";
    ctx.style.top = Math.min(e.clientY, window.innerHeight - ctx.offsetHeight - 15) + "px";

    anime({
        targets: ctx,
        rotateX: 0,
        skewY: 0,
        opacity: 1,
        duration: 1000,
    }).finished.then(() => {
        document.getElementById('c-m-d-upload-file').onclick = () => {
            globs.file_up_wr.style.display = 'block';
            showModal('.file-upload-container', '.file-upload-container-wrapper');
        };

        document.getElementById('c-m-d-create-dir').onclick = () => {
            let ndname = window.prompt("Enter new name: ");
            if (ndname === "" || ndname === null)
                return;

            fetch('/api/rec/dir', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        parentDir: did,
                        dirName: ndname,
                    }),
                })
                .then(res => {
                    if (res.ok)
                        return res.json();
                })
                .then(res => {
                    if (!res)
                        return;
                    loadDir(did);
                });
        };
    });
}

function hideContextMenus() {
    if (globs.item_ctx_menu.style.opacity !== '0') {
        anime({
            targets: globs.item_ctx_menu,
            rotateX: 90,
            skewY: 45,
            opacity: 0,
            duration: 100,
            easing: 'linear',
        }).finished.then(() => {
            globs.item_ctx_menu.style.display = 'none';
        });
    }
    if (globs.dir_ctx_menu.style.opacity !== '0') {
        anime({
            targets: globs.dir_ctx_menu,
            rotateX: 90,
            skewY: 45,
            opacity: 0,
            duration: 100,
            easing: 'linear',
        }).finished.then(() => {
            globs.dir_ctx_menu.style.display = 'none';
        });
    }
}

function hideModal(modal_cl, modal_wr_cl) {
    anime({
        targets: modal_cl,
        translateY: -500,
        opacity: 0,
        duration: 750,
    });
    anime({
        targets: modal_wr_cl,
        opacity: 0,
        duration: 150,
        easing: 'linear',
    }).finished.then(() => {
        document.getElementsByClassName(modal_wr_cl.substr(1))[0].style.display = 'none';
    });
}

function showModal(modal_cl, modal_wr_cl) {
    anime({
        targets: modal_cl,
        translateY: 0,
        opacity: 1,
        duration: 1000,
    });
    anime({
        targets: modal_wr_cl,
        opacity: 1,
        duration: 500,
    });
}

function uploadFile(t_file, prog = document.getElementById('file-upload-upload-progress'),
    fname = document.getElementById('file-upload-upload-filename'), cb = () => {}) {

    let freader = new FileReader();
    return new Promise((resolve, reject) => {
        freader.onload = function () {
            let i_a = new Uint8Array(freader.result);

            fname.innerHTML = `encrypting "${t_file.name}"`;
            fname.classList.add('file-upload-encrypting');
            fname.classList.remove('file-upload-uploading');
            prog.removeAttribute('value');

            let bts = marlx_crypto.hexUndigest(getCookie('marlx_scr'));
            let k = marlx_crypto.stringToBytes(getCookie('marlx_tkn').substr(0, 32));
            marlx_crypto.aesctrDec(bts, k).then(pwd => {
                pwd = marlx_crypto.bytesToString(pwd);
                let salt = marlx_crypto.genSalt();

                marlx_crypto.hash(pwd, salt).then(h => {
                    let key = marlx_crypto.stringToBytes(h.substr(28));

                    marlx_crypto.aesctrEnc(i_a, key).then(enc => {
                        let fd = new FormData();
                        let f = new File([enc.buffer], t_file.name, {
                            type: t_file.type,
                            lastModified: t_file.lastModified
                        });

                        fd.append('file', f, t_file.name);
                        fname.innerHTML = `uploading "${t_file.name}"`;
                        fname.classList.remove('file-upload-encrypting');
                        fname.classList.add('file-upload-uploading');

                        var xhr = new XMLHttpRequest();
                        xhr.onreadystatechange = function () {
                            if (this.readyState === 4) {
                                console.log(this.responseText);
                            }
                            if (this.readyState === 4 && this.status === 200) {
                                cb();
                                resolve(this.responseText);
                                prog.value = 0;
                                fname.innerHTML = '';
                            }
                        };
                        xhr.upload.onprogress = function (e) {
                            prog.value = e.loaded;
                            prog.max = e.total;
                        };
                        xhr.open('POST',
                            `/api/rec/file?parDir=${globs.cu_dir}&salt=${salt}&osize=${t_file.size}&type=${t_file.type||'unknown'}`
                        );
                        xhr.send(fd);
                    });
                });
            });
        };
        freader.onprogress = function (e) {
            prog.value = e.loaded;
            prog.max = e.total;
        };
        // freader.onerror = function () {
        //     console.log('some error ... ');
        // };
        freader.readAsArrayBuffer(t_file);
    });
}

function refreshUploadFiles() {
    let trgt = document.getElementsByClassName('file-upload-drop-area-files')[0];
    trgt.innerHTML = '';

    let genFEl = (name, size) => {
        let d = document.createElement('div');
        d.classList.add('file-upload-file-item');
        d.innerHTML = `
            <span class="file-upload-file-name">${name}</span>
            <span class="file-upload-file-size">${parseSize(size)}</span>
        `;
        return d;
    };

    if (globs.up_files.length === 0 && globs.up_files_in.length === 0) {
        document.getElementsByClassName('file-upload-drop-here')[0].style.display = 'flex';
        return;
    }
    document.getElementsByClassName('file-upload-drop-here')[0].style.display = 'none';

    [...globs.up_files, ...globs.up_files_in].forEach(f => {
        trgt.appendChild(genFEl(f.name, f.size));
    });
}

function refreshClients() {
    return fetch('/api/clnts/list')
        .then(res => {
            if (res.ok)
                return res.json();
        })
        .then(res => {
            if (!res)
                return [];

            globs.u_clients = res;
            return res;
        });
}

function getClientInfo(hex_tkn) {
    return fetch(`/api/clnts/info?tkn=${hex_tkn}`)
        .then(res => {
            if (res.ok)
                return res.json();
            return {};
        });
}

function showClientsInfo(trgt = document.getElementById('client-info-clients-out')) {
    refreshClients()
        .then(clients => {
            trgt.innerHTML = '';
            clients.forEach(c => {
                getClientInfo(c).then(ci => {
                    let d = document.createElement('div');
                    d.classList.add('client-info-element');

                    let h = document.createElement('h3');
                    h.id = `c-i-client-${ci.Token}`;
                    h.classList.add('client-info-hostname');
                    h.innerHTML = ci.Hostname;
                    d.appendChild(h);

                    let t = document.createElement('table');
                    t.classList.add('client-info-table');
                    d.appendChild(t);

                    let tr = document.createElement('tr');
                    t.appendChild(tr);

                    let td = document.createElement('td');
                    tr.appendChild(td);
                    td.innerHTML = 'Token:';

                    td = document.createElement('td');
                    tr.appendChild(td);
                    td.innerHTML = c;

                    tr = document.createElement('tr');
                    t.appendChild(tr);

                    td = document.createElement('td');
                    tr.appendChild(td);
                    td.innerHTML = 'MTU:';

                    td = document.createElement('td');
                    tr.appendChild(td);
                    td.innerHTML = parseSize(ci.MTU);

                    let con = document.createElement('div');
                    con.classList.add('client-info-storage-stats');
                    d.appendChild(con);

                    let can = document.createElement('canvas');
                    can.classList.add('client-info-storage-stats-canvas');
                    con.appendChild(can);

                    new Chart(can, {
                        type: 'doughnut',
                        data: {
                            datasets: [{
                                data: [
                                    ci.TotalBytes - ci.FreeBytes,
                                    ci.FreeBytes,
                                ],
                                backgroundColor: [
                                    '#FF8484',
                                    '#AFFFB7',
                                ],
                                label: 'Storage Space',
                            }],
                            labels: [
                                'Used',
                                'Free'
                            ],
                        },
                        options: {
                            responsive: true,
                            legend: {
                                position: 'top',
                            },
                            title: {
                                display: true,
                                text: `${ci.Hostname} - Disk Information`,
                            },
                            animation: {
                                animateScale: true,
                                animateRotate: true,
                            },
                            tooltips: {
                                callbacks: {
                                    label: (tooltipItem, data) => {
                                        let lbl = data.labels[tooltipItem.index] || '';
                                        if (lbl)
                                            lbl += ': ';

                                        lbl += parseSize(data.datasets[tooltipItem.datasetIndex].data[tooltipItem.index]);
                                        return lbl;
                                    },
                                },
                            },
                        },
                    });

                    trgt.appendChild(d);
                });
            });

            if (clients.length === 0) {
                trgt.appendChild(document.createTextNode('No clients have been created yet ... '));
            }
        });
}

function showClientInfo(tkn, trgt = document.getElementById('client-info-clients-out')) {
    globs.client_inf_wr.style.display = 'block';
    showModal('.client-info-container', '.client-info-container-wrapper');
    showClientsInfo(trgt);
    document.getElementById(`c-i-client-${tkn}`).scrollIntoView(true);
}

function createClient(cb = () => {}) {
    fetch('/api/clnts/new')
        .then(res => {
            if (res.ok)
                return res.text();
        })
        .then(res => {
            if (!res)
                return;
            cb();
        });
}

// --------------- //

// -- INITIALISATION -- //

function listenerInit() {
    document.onclick = () => {
        hideContextMenus();
    };
    document.oncontextmenu = () => {
        hideContextMenus();
    };
    document.getElementById('i-i-p-close').onclick = () => {
        hideModal('.item-info-popup', '.item-info-popup-wrapper');
    };
    document.getElementsByClassName('main-content')[0].oncontextmenu = e => {
        e.preventDefault();
        if (e.target.classList.contains('dir-trace') || e.target.classList.contains('dir-trace-item') ||
            e.target.parentNode.classList.contains('dir-trace-item') || e.target.classList.contains('dir-item') ||
            e.target.parentNode.classList.contains('dir-item') || e.target.classList.contains('dir-item-preview') ||
            e.target.parentNode.classList.contains('dir-item-preview') || e.target.classList.contains('dir-item-info') ||
            e.target.parentNode.classList.contains('dir-item-info')) {
            return;
        }

        displayDirContextMenu(e);
    };
    document.getElementById('f-u-close').onclick = () => {
        hideModal('.file-upload-container', '.file-upload-container-wrapper');
        globs.up_files = [];
        globs.up_files_in = [];
        refreshUploadFiles();
    };
    document.getElementById('f-p-close').onclick = () => {
        hideModal('.file-preview-container', '.file-preview-wrapper');
        refreshUploadFiles();
    };
    document.getElementsByClassName('file-upload-drop-area')[0].ondragover = e => {
        e.preventDefault();
    };
    document.getElementsByClassName('file-upload-drop-area')[0].ondrop = e => {
        e.preventDefault();

        if (e.dataTransfer.items) {
            new Array(...e.dataTransfer.items).forEach(i => {
                if (i.kind === 'file') {
                    let file = i.getAsFile();
                    globs.up_files.push(file);
                }
            });
        } else {
            new Array(...e.dataTransfer.files).forEach(f => {
                globs.up_files.push(f);
            });
        }

        refreshUploadFiles();
    };
    document.getElementsByClassName('main-content')[0].ondragover = e => {
        // e.preventDefault();
        // if (e.dataTransfer.files) {
        //     globs.file_up_wr.style.display = 'block';
        //     showModal('.file-upload-container', '.file-upload-container-wrapper');
        // }
    };
    document.getElementsByClassName('file-upload-drop-area')[0].onclick = () => {
        document.getElementById('file-upload-hidden-in').click();
    };
    document.getElementById('file-upload-hidden-in').onchange = () => {
        globs.up_files_in = [...document.getElementById('file-upload-hidden-in').files];
        refreshUploadFiles();
    };
    document.getElementById('file-upload-submit').onclick = () => {
        (async function () {
            for (let t_file of globs.up_files) {
                await uploadFile(t_file);
            }

            for (let i = 0; i < globs.up_files_in.length; i++) {
                await uploadFile(globs.up_files_in[i]);
            }

            if (globs.up_files.length === 0)
                document.getElementById('f-u-close').click();

            globs.up_files = [];
            globs.up_files_in = [];
            document.getElementById('file-upload-hidden-in').value = '';

            document.getElementById('f-u-close').click();
            loadDir(globs.cu_dir);

            refreshUploadFiles();
        })();
    };
    document.getElementById('u-p-d-clients').onclick = () => {
        globs.client_inf_wr.style.display = 'block';
        showModal('.client-info-container', '.client-info-container-wrapper');
        showClientsInfo();
    };
    document.getElementById('c-i-close').onclick = () => {
        hideModal('.client-info-container', '.client-info-container-wrapper');
    };
    document.getElementById('client-info-creation-button').onclick = () => {
        createClient(showClientInfo);
    };
    document.getElementById('user-profile').onclick = () => {
        let trgt = document.getElementById('user-profile-dropdown');

        if (trgt.style.display === 'block') {
            anime({
                targets: trgt,
                opacity: 0,
                easing: 'linear',
                duration: 75,
            }).finished.then(() => {
                trgt.style.display = 'none';
            });
        } else {
            trgt.style.display = 'block';
            anime({
                targets: trgt,
                opacity: 1,
                easing: 'linear',
                duration: 100,
            });
        }
    };
}

function init() {
    getUserCreds();
    loadDir(window.location.pathname.substr(1));
    listenerInit();
}

window.onload = () => {
    init();
};

// -------------------- //