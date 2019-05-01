const bcrypt = require('bcryptjs');
const aesjs = require('aes-js');

module.exports.sizeof = function sizeof(str) {
    str = String(str);

    let blen = 0;
    for (let i = 0; i < str.length; i++) {
        let c = str.charCodeAt(i);
        blen += c < (1 << 7)  ? 1 :
                c < (1 << 11) ? 2 :
                c < (1 << 16) ? 3 :
                c < (1 << 21) ? 4 :
                c < (1 << 26) ? 5 :
                c < (1 << 31) ? 6 : Number.NaN;
    }
    return blen;
}

module.exports.aesctrStrEnc = function aesctrStrEnc(str, key) {
    return module.exports.aesctrEnc(aesjs.utils.utf8.toBytes(str), key);
}

module.exports.aesctrEnc = function aesctrEnc(plain, key) {
    let aesCtr = new aesjs.ModeOfOperation.ctr(key);
    return new Promise(res => res(aesCtr.encrypt(plain)));
}

module.exports.aesctrStrDec = function aesctrStrDec(enc, key) {
    return aesjs.utils.utf8.fromBytes(module.exports.aesctrDec(enc, key));
}

module.exports.aesctrDec = function aesctrDec(enc, key) {
    let aesCtr = new aesjs.ModeOfOperation.ctr(key);
    return new Promise(res => res(aesCtr.decrypt(enc)));
}

module.exports.hexDigest = function hexDigest(dat) {
    return aesjs.utils.hex.fromBytes(dat);
}

module.exports.hexUndigest = function hexUndigest(hex) {
    return aesjs.utils.hex.toBytes(hex);
}

module.exports.bytesToString = function bytesToString(bts) {
    return aesjs.utils.utf8.fromBytes(bts);
}

module.exports.stringToBytes = function stringToBytes(str) {
    return aesjs.utils.utf8.toBytes(str);
}

module.exports.genSalt = function genSalt(rounds=12) {
    return bcrypt.genSaltSync(rounds);
}

module.exports.hash = function hash(str, salt=module.exports.genSalt()) {
    return bcrypt.hash(str, salt);
}