fetch('graphs/server_data.json')
    .then(response => response.json())
    .then(data => {
        let nodes = [];
        let links = [];
        
        data.forEach((server, index) => {
            // Create a node for the server
            nodes.push({
                id: server.Hostname,
                group: index,
                type: 'server'
            });
        
            server.Connections.forEach(connection => {
                // Create a node for the port
                let portId = `${server.Hostname}:${connection.Port}`;
                nodes.push({
                    id: portId,
                    group: index,
                    type: 'port'
                });
        
                // Create a link from the server to the port
                links.push({
                    source: server.Hostname,
                    target: portId,
                    value: connection.Success ? 1 : 0
                });
            });
        });

        console.log(nodes)
        console.log(links)

        width = 960;
        height = 600;

        // Create the D3.js force-directed graph
        let svg = d3.select("body").append("svg")
            .attr("width", width)
            .attr("height", height);

        let simulation = d3.forceSimulation(nodes)
            .force("link", d3.forceLink(links).id(d => d.id))
            .force("charge", d3.forceManyBody())
            .force("center", d3.forceCenter(width / 2, height / 2));

        // Add the links
        let link = svg.append("g")
            .attr("class", "links")
            .selectAll("line")
            .data(links)
            .enter().append("line")
            .attr("stroke", "#999") // Add this line            
            .attr("stroke-width", d => 10)
            // .attr("stroke-width", d => Math.sqrt(d.value*100))
            .on("mouseover", function (d) {
                tooltip.transition()
                    .duration(200)
                    .style("opacity", .9);
                tooltip.html("Source: " + d.source.id + "<br/>" + "Target: " + d.target.id + "<br/>" + "Port: " + d.port + "<br/>" + "Success: " + (d.value ? "Yes" : "No"))
                    .style("left", (d3.event.pageX) + "px")
                    .style("top", (d3.event.pageY - 28) + "px");
            })
            .on("mouseout", function (d) {
                tooltip.transition()
                    .duration(500)
                    .style("opacity", 0);
            });

        // Define the tooltip
        let tooltip = d3.select("body").append("div")
            .attr("class", "tooltip")
            .style("opacity", 0);

        let color = d3.scaleOrdinal(d3.schemeCategory10);

        // Add the nodes
        svg.append("g")
            .attr("class", "nodes")
            .selectAll("circle")
            .data(nodes)
            .enter().append("circle")
            .attr("r", 100)
            .attr("fill", d => color(d.group))
            .call(d3.drag()
                .on("start", dragstarted)
                .on("drag", dragged)
                .on("end", dragended));

        // Add the node labels
        let labels = svg.append("g")
            .attr("class", "labels")
            .selectAll("text")
            .data(nodes)
            .enter().append("text")
            .attr("dx", 12)
            .attr("dy", ".35em")
            .text(d => d.id);

        // Define the drag event functions
        function dragstarted(event, d) {
            if (!event.active) simulation.alphaTarget(0.3).restart();
            d.fx = d.x;
            d.fy = d.y;
        }

        function dragged(event, d) {
            d.fx = event.x;
            d.fy = event.y;
        }

        function dragended(event, d) {
            if (!event.active) simulation.alphaTarget(0);
            d.fx = null;
            d.fy = null;
        }

        simulation.on("tick", () => {
            svg.selectAll("line")
                .attr("x1", d => d.source.x)
                .attr("y1", d => d.source.y)
                .attr("x2", d => d.target.x)
                .attr("y2", d => d.target.y);

            svg.selectAll("circle")
                .attr("cx", d => d.x)
                .attr("cy", d => d.y);

            labels
                .attr("x", d => d.x)
                .attr("y", d => d.y);
        });
    })
    .catch(error => console.error('Error:', error));

