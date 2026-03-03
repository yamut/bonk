(function () {
    const canvas = document.getElementById('canvas');
    const ctx = canvas.getContext('2d');
    const W = 800, H = 600;
    const PADDLE_W = 10, PADDLE_H = 80, PADDLE_OFF = 20, BALL_SIZE = 10;

    let ws;
    let state = null;
    let mySide = null;
    let lastDir = 0;
    let inQueue = false;

    // Connect WebSocket
    function connect() {
        const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
        ws = new WebSocket(proto + '//' + location.host + '/ws');

        ws.onmessage = function (e) {
            const env = JSON.parse(e.data);
            switch (env.type) {
                case 'lobby':
                    document.getElementById('waiting').textContent =
                        env.data.waiting + ' player(s) waiting';
                    document.getElementById('queue-waiting').textContent =
                        env.data.waiting + ' player(s) in queue';
                    break;
                case 'start':
                    mySide = env.data.side;
                    inQueue = false;
                    state = null;
                    document.getElementById('lobby').style.display = 'none';
                    document.getElementById('queue').style.display = 'none';
                    document.getElementById('game').style.display = 'block';
                    document.getElementById('info').textContent =
                        'You are ' + mySide + ' | vs ' + env.data.opponent;
                    break;
                case 'state':
                    state = env.data;
                    break;
                case 'frame':
                    if (!state) {
                        state = {
                            ball: { x: env.data.bx, y: env.data.by },
                            left_paddle: { y: env.data.lp },
                            right_paddle: { y: env.data.rp },
                            left_score: 0,
                            right_score: 0,
                        };
                    } else {
                        state.ball.x = env.data.bx;
                        state.ball.y = env.data.by;
                        state.left_paddle.y = env.data.lp;
                        state.right_paddle.y = env.data.rp;
                    }
                    break;
                case 'over':
                    document.getElementById('game').style.display = 'none';
                    document.getElementById('game-over').style.display = 'block';
                    const won = env.data.winner === mySide;
                    document.getElementById('result').textContent =
                        won ? 'YOU WIN!' : 'YOU LOSE';
                    break;
            }
        };

        ws.onclose = function () {
            if (!state || !state.over) {
                document.getElementById('lobby').style.display = 'none';
                document.getElementById('queue').style.display = 'none';
                document.getElementById('game').style.display = 'none';
                document.getElementById('game-over').style.display = 'block';
                document.getElementById('result').textContent = 'DISCONNECTED';
            }
        };
    }

    // Join game
    window.join = function (mode) {
        if (inQueue) return;
        const name = document.getElementById('name').value.trim() || 'Anonymous';
        ws.send(JSON.stringify({
            type: 'join',
            data: { name: name, mode: mode }
        }));
        if (mode === 'pvp') {
            inQueue = true;
            document.getElementById('lobby').style.display = 'none';
            document.getElementById('queue').style.display = 'block';
        }
    };

    // Cancel queue
    window.cancelQueue = function () {
        ws.send(JSON.stringify({ type: 'leave', data: {} }));
        inQueue = false;
        document.getElementById('queue').style.display = 'none';
        document.getElementById('lobby').style.display = 'block';
    };

    // Input handling — send direction changes only
    const keys = {};
    document.addEventListener('keydown', function (e) {
        if (keys[e.key]) return;
        keys[e.key] = true;
        sendDirection();
    });
    document.addEventListener('keyup', function (e) {
        delete keys[e.key];
        sendDirection();
    });

    function sendDirection() {
        if (!ws || ws.readyState !== WebSocket.OPEN) return;
        let dir = 0;
        if (keys['ArrowUp'] || keys['w']) dir -= 1;
        if (keys['ArrowDown'] || keys['s']) dir += 1;
        if (dir !== lastDir) {
            lastDir = dir;
            ws.send(JSON.stringify({
                type: 'input',
                data: { direction: dir }
            }));
        }
    }

    // Touch input — follow finger
    let touchY = null;

    canvas.addEventListener('touchstart', function (e) {
        e.preventDefault();
        const rect = canvas.getBoundingClientRect();
        touchY = (e.touches[0].clientY - rect.top) / rect.height * H;
    }, { passive: false });

    canvas.addEventListener('touchmove', function (e) {
        e.preventDefault();
        const rect = canvas.getBoundingClientRect();
        touchY = (e.touches[0].clientY - rect.top) / rect.height * H;
    }, { passive: false });

    canvas.addEventListener('touchend', function (e) {
        e.preventDefault();
        touchY = null;
        if (lastDir !== 0) {
            lastDir = 0;
            ws.send(JSON.stringify({
                type: 'input',
                data: { direction: 0 }
            }));
        }
    }, { passive: false });

    // Rendering
    function draw() {
        // Touch follow — send direction toward finger each frame
        if (touchY !== null && state && ws && ws.readyState === WebSocket.OPEN) {
            const paddleY = mySide === 'left' ? state.left_paddle.y : state.right_paddle.y;
            const deadZone = 10;
            let dir = 0;
            if (touchY < paddleY - deadZone) dir = -1;
            else if (touchY > paddleY + deadZone) dir = 1;
            if (dir !== lastDir) {
                lastDir = dir;
                ws.send(JSON.stringify({
                    type: 'input',
                    data: { direction: dir }
                }));
            }
        }

        ctx.fillStyle = '#111';
        ctx.fillRect(0, 0, W, H);

        if (!state) {
            requestAnimationFrame(draw);
            return;
        }

        // Center line
        ctx.setLineDash([10, 10]);
        ctx.strokeStyle = '#333';
        ctx.lineWidth = 2;
        ctx.beginPath();
        ctx.moveTo(W / 2, 0);
        ctx.lineTo(W / 2, H);
        ctx.stroke();
        ctx.setLineDash([]);

        // Scores
        ctx.fillStyle = '#fff';
        ctx.font = '48px Courier New';
        ctx.textAlign = 'center';
        ctx.fillText(state.left_score, W / 4, 60);
        ctx.fillText(state.right_score, 3 * W / 4, 60);

        // Left paddle
        ctx.fillStyle = '#fff';
        ctx.fillRect(
            PADDLE_OFF,
            state.left_paddle.y - PADDLE_H / 2,
            PADDLE_W,
            PADDLE_H
        );

        // Right paddle
        ctx.fillRect(
            W - PADDLE_OFF - PADDLE_W,
            state.right_paddle.y - PADDLE_H / 2,
            PADDLE_W,
            PADDLE_H
        );

        // Ball
        ctx.fillRect(
            state.ball.x - BALL_SIZE / 2,
            state.ball.y - BALL_SIZE / 2,
            BALL_SIZE,
            BALL_SIZE
        );

        requestAnimationFrame(draw);
    }

    connect();
    requestAnimationFrame(draw);
})();
