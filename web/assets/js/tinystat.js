$('.ui.dropdown').dropdown();

function pollStats() {
    $.getJSON('/v1/stats', function (data) {
        // Set all countups
        countup('actionsRecorded', data.actionsRecorded);
        countup('countsCalculated', data.countsCalculated);
        countup('summariesCalculated', data.summariesCalculated);

        // Perform this action every 10 seconds
        setTimeout(pollStats, 10000); // Poll stats every 10 seconds and re-apply to UI
    });
}

// countup animates the passed id with a new counted up value
function countup(id, to, prefix, suffix) {
    var from = $('#' + id).text(); // Retrieve the starting value
    from = from.trim(); // Trim any whitespace
    from = from.replace(/,/g, ''); // Remove all commas
    if (from == '' || from > to) {
        from = to;
    }

    // Configure countup options
    var options = { 
        useEasing: false, 
        useGrouping: true, 
        separator: ',', 
        decimal: '.'
    };

    // Apply a prefix if one is passed
    if (prefix != undefined && prefix != '') {
        options.prefix = prefix;
    }

    // Apply a suffix if one is passed
    if (suffix != undefined && suffix != '') {
        options.suffix = suffix;
    }

    // Trigger the countup animation
    var count = new CountUp(id, from, to, 0, 10, options);
    if (!count.error) {
        count.start();
    } else {
        console.error(count.error);
    }
}

$(document).ready(function () {
    pollStats();
    $('#new-app-form').on('submit', function (e) {
        e.preventDefault();
        var name = document.getElementById('app-name').value;
        var secure = document.getElementById('app-security').value;

        // Verify the parameters were passed
        if (name === '' || secure === '') {
            return;
        }

        // Set the loading button
        $('#new-app-button').addClass('loading');

        // Build the request URL
        var url = '/v1/app/create/' + name;
        if (secure == 'true') {
            url = url + '?strict_auth=true';
        }

        // Perform the get request
        $.post(url, function (data) {
            alert('AppID: ' + data.id + '\n' + 'AppToken: ' + data.token);
            $('#new-app-button').removeClass('loading');
        }, 'json');
    });
});