node {
    checkout scm
        
    stage('Docker Build') {
        docker.build('dukfaar/itembackend')
    }

    stage('Update Service') {
        sh 'docker service update --force itembackend_itembackend'
    }
}
